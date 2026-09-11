// Package services holds the product-module application logic: products and
// their variants, images and category links.
package services

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	attrdomain "github.com/jayedbinnazir/product-service/internal/attributes/domain"
	attributesrepo "github.com/jayedbinnazir/product-service/internal/attributes/repository"
	categoryrepo "github.com/jayedbinnazir/product-service/internal/category/repository"
	objstore "github.com/jayedbinnazir/product-service/internal/infrastructure/s3"
	"github.com/jayedbinnazir/product-service/internal/platform"
	"github.com/jayedbinnazir/product-service/internal/product/domain"
	"github.com/jayedbinnazir/product-service/internal/product/dto"
	"github.com/jayedbinnazir/product-service/internal/product/repository"
)

var slugRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

const uploadURLTTL = 15 * time.Minute

type Service struct {
	db      *sql.DB
	storage *objstore.Storage
}

func New(db *sql.DB, storage *objstore.Storage) *Service {
	return &Service{db: db, storage: storage}
}

// ---------------------------------------------------------------------
// Products
// ---------------------------------------------------------------------

func (s *Service) CreateProduct(ctx context.Context, tenantID uuid.UUID, req dto.CreateProductRequest) (*domain.Detail, error) {
	slug := normalizeSlug(req.Slug)
	if !slugRE.MatchString(slug) {
		return nil, domain.ErrInvalidSlug
	}

	product := &domain.Product{
		TenantID:    tenantID,
		Name:        strings.TrimSpace(req.Name),
		Slug:        slug,
		Description: trimPtr(req.Description),
		Status:      domain.StatusDraft,
	}
	if err := repository.NewProductRepository(s.db).Create(ctx, product); err != nil {
		return nil, err
	}
	return s.detail(ctx, s.db, product)
}

// GetProduct returns a product of any status (management view).
func (s *Service) GetProduct(ctx context.Context, tenantID, id uuid.UUID) (*domain.Detail, error) {
	product, err := repository.NewProductRepository(s.db).GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, s.db, product)
}

// GetPublishedProduct returns a product only if it is ACTIVE (storefront view).
func (s *Service) GetPublishedProduct(ctx context.Context, tenantID, id uuid.UUID) (*domain.Detail, error) {
	product, err := repository.NewProductRepository(s.db).GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if product.Status != domain.StatusActive {
		return nil, domain.ErrProductNotFound
	}
	return s.detail(ctx, s.db, product)
}

func (s *Service) ListProducts(ctx context.Context, f domain.ListFilter) ([]domain.Product, error) {
	return repository.NewProductRepository(s.db).List(ctx, f)
}

func (s *Service) UpdateProduct(ctx context.Context, tenantID, id uuid.UUID, req dto.UpdateProductRequest) (*domain.Detail, error) {
	if req.IsEmpty() {
		return nil, domain.ErrNoUpdateFields
	}

	productRepo := repository.NewProductRepository(s.db)
	product, err := productRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		product.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil {
		slug := normalizeSlug(*req.Slug)
		if !slugRE.MatchString(slug) {
			return nil, domain.ErrInvalidSlug
		}
		product.Slug = slug
	}
	if req.Description != nil {
		product.Description = trimPtr(req.Description)
	}
	if req.Status != nil {
		status := domain.Status(*req.Status)
		if !status.Valid() {
			return nil, domain.ErrInvalidStatus
		}
		if status == domain.StatusActive {
			count, err := repository.NewVariantRepository(s.db).CountByProduct(ctx, product.ID)
			if err != nil {
				return nil, err
			}
			if count == 0 {
				return nil, domain.ErrActivateNeedsVariant
			}
		}
		product.Status = status
	}

	if err := productRepo.Update(ctx, product); err != nil {
		return nil, err
	}
	return s.detail(ctx, s.db, product)
}

func (s *Service) DeleteProduct(ctx context.Context, tenantID, id uuid.UUID) error {
	return repository.NewProductRepository(s.db).Delete(ctx, tenantID, id)
}

// SetCategories replaces a product's category links after checking every
// category belongs to the same tenant.
func (s *Service) SetCategories(ctx context.Context, tenantID, productID uuid.UUID, rawIDs []string) ([]uuid.UUID, error) {
	if _, err := repository.NewProductRepository(s.db).GetByID(ctx, tenantID, productID); err != nil {
		return nil, err
	}

	// parse + dedupe the incoming ids
	seen := make(map[uuid.UUID]struct{}, len(rawIDs))
	ids := make([]uuid.UUID, 0, len(rawIDs))
	for _, raw := range rawIDs {
		id := uuid.MustParse(raw)
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	// one query validates the whole set belongs to this tenant (no N+1)
	if len(ids) > 0 {
		count, err := categoryrepo.New(s.db).CountExisting(ctx, tenantID, ids)
		if err != nil {
			return nil, err
		}
		if count != len(ids) {
			return nil, domain.ErrCategoryNotInTenant
		}
	}

	if err := repository.NewCategoryLinkRepository(s.db).SetForProduct(ctx, productID, ids); err != nil {
		return nil, err
	}
	return ids, nil
}

// ---------------------------------------------------------------------
// Variants
// ---------------------------------------------------------------------

func (s *Service) AddVariant(ctx context.Context, tenantID, productID uuid.UUID, req dto.CreateVariantRequest) (*domain.Variant, error) {
	if _, err := repository.NewProductRepository(s.db).GetByID(ctx, tenantID, productID); err != nil {
		return nil, err
	}

	variant := &domain.Variant{
		ProductID:           productID,
		TenantID:            tenantID,
		SKU:                 strings.TrimSpace(req.SKU),
		Title:               trimPtr(req.Title),
		PriceCents:          req.PriceCents,
		Currency:            currencyOrDefault(req.Currency),
		CompareAtPriceCents: req.CompareAtPriceCents,
		WeightGrams:         req.WeightGrams,
		Barcode:             trimPtr(req.Barcode),
		Position:            req.Position,
		IsDefault:           req.IsDefault,
	}

	err := platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		opts, err := s.resolveVariantOptions(ctx, tx, productID, req.Options)
		if err != nil {
			return err
		}

		repo := repository.NewVariantRepository(tx)
		if variant.IsDefault {
			if err := repo.ClearDefault(ctx, productID); err != nil {
				return err
			}
		}
		if err := repo.Create(ctx, variant); err != nil {
			return err
		}
		if len(opts) == 0 {
			return nil
		}
		for i := range opts {
			opts[i].VariantID = variant.ID
		}
		return repository.NewVariantOptionRepository(tx).SetForVariant(ctx, variant.ID, opts)
	})
	if err != nil {
		return nil, err
	}
	return variant, nil
}

func (s *Service) UpdateVariant(ctx context.Context, tenantID, productID, variantID uuid.UUID, req dto.UpdateVariantRequest) (*domain.Variant, error) {
	if req.IsEmpty() {
		return nil, domain.ErrNoUpdateFields
	}

	var updated *domain.Variant
	err := platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		repo := repository.NewVariantRepository(tx)

		variant, err := repo.GetByID(ctx, variantID)
		if err != nil {
			return err
		}
		if variant.ProductID != productID || variant.TenantID != tenantID {
			return domain.ErrVariantNotInProduct
		}

		applyVariantPatch(variant, req)

		if variant.IsDefault {
			if err := repo.ClearDefault(ctx, productID); err != nil {
				return err
			}
		}
		if err := repo.Update(ctx, variant); err != nil {
			return err
		}

		if req.Options != nil { // nil = leave as-is; [] = clear
			opts, err := s.resolveVariantOptions(ctx, tx, productID, req.Options)
			if err != nil {
				return err
			}
			for i := range opts {
				opts[i].VariantID = variant.ID
			}
			if err := repository.NewVariantOptionRepository(tx).SetForVariant(ctx, variant.ID, opts); err != nil {
				return err
			}
		}

		updated = variant
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) DeleteVariant(ctx context.Context, tenantID, productID, variantID uuid.UUID) error {
	repo := repository.NewVariantRepository(s.db)
	variant, err := repo.GetByID(ctx, variantID)
	if err != nil {
		return err
	}
	if variant.ProductID != productID || variant.TenantID != tenantID {
		return domain.ErrVariantNotInProduct
	}
	return repo.Delete(ctx, variantID)
}

// ---------------------------------------------------------------------
// Images
// ---------------------------------------------------------------------

// ImageUpload is the presigned-URL ticket returned to the client.
type ImageUpload struct {
	ImageID    uuid.UUID
	StorageKey string
	UploadURL  string
	PublicURL  string
	ExpiresIn  int
}

// CreateImageUploadURL verifies the product and returns a presigned URL the
// client PUTs the raw file to. Nothing is stored until ConfirmImage.
func (s *Service) CreateImageUploadURL(ctx context.Context, tenantID, productID uuid.UUID, contentType string) (*ImageUpload, error) {
	if _, err := repository.NewProductRepository(s.db).GetByID(ctx, tenantID, productID); err != nil {
		return nil, err
	}

	imageID := uuid.New()
	key, _, ok := s.storage.NewImageKey(tenantID, productID, imageID, contentType)
	if !ok {
		return nil, domain.ErrUnsupportedImageType
	}

	uploadURL, err := s.storage.PresignPut(ctx, key, contentType, uploadURLTTL)
	if err != nil {
		return nil, err
	}

	return &ImageUpload{
		ImageID:    imageID,
		StorageKey: key,
		UploadURL:  uploadURL,
		PublicURL:  s.storage.PublicURL(key),
		ExpiresIn:  int(uploadURLTTL.Seconds()),
	}, nil
}

// ConfirmImage registers an object the client has finished uploading.
func (s *Service) ConfirmImage(ctx context.Context, tenantID, productID uuid.UUID, req dto.ConfirmImageRequest) (*domain.Image, error) {
	if _, err := repository.NewProductRepository(s.db).GetByID(ctx, tenantID, productID); err != nil {
		return nil, err
	}

	key := strings.TrimSpace(req.StorageKey)
	if !s.storage.KeyBelongsTo(key, tenantID, productID) {
		return nil, domain.ErrImageKeyMismatch
	}

	exists, err := s.storage.Exists(ctx, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrImageNotUploaded
	}

	img := &domain.Image{
		ID:         imageIDFromKey(key),
		ProductID:  productID,
		URL:        s.storage.PublicURL(key),
		StorageKey: key,
		Alt:        trimPtr(req.Alt),
		Position:   req.Position,
	}
	if err := repository.NewImageRepository(s.db).Create(ctx, img); err != nil {
		return nil, err
	}
	return img, nil
}

func (s *Service) UpdateImage(ctx context.Context, tenantID, productID, imageID uuid.UUID, req dto.UpdateImageRequest) (*domain.Image, error) {
	if req.IsEmpty() {
		return nil, domain.ErrNoUpdateFields
	}
	if _, err := repository.NewProductRepository(s.db).GetByID(ctx, tenantID, productID); err != nil {
		return nil, err
	}

	repo := repository.NewImageRepository(s.db)
	img, err := repo.GetByID(ctx, imageID)
	if err != nil {
		return nil, err
	}
	if img.ProductID != productID {
		return nil, domain.ErrImageNotFound
	}

	if req.Alt != nil {
		img.Alt = trimPtr(req.Alt)
	}
	if req.Position != nil {
		img.Position = *req.Position
	}

	if err := repo.Update(ctx, img); err != nil {
		return nil, err
	}
	return img, nil
}

// DeleteImage removes the row and the object in S3.
func (s *Service) DeleteImage(ctx context.Context, tenantID, productID, imageID uuid.UUID) error {
	if _, err := repository.NewProductRepository(s.db).GetByID(ctx, tenantID, productID); err != nil {
		return err
	}
	repo := repository.NewImageRepository(s.db)
	img, err := repo.GetByID(ctx, imageID)
	if err != nil {
		return err
	}
	if img.ProductID != productID {
		return domain.ErrImageNotFound
	}
	if err := repo.Delete(ctx, imageID); err != nil {
		return err
	}
	// Best-effort: the row is gone; a leftover object is harmless.
	_ = s.storage.Delete(ctx, img.StorageKey)
	return nil
}

// imageIDFromKey pulls the <image_id> out of products/<tenant>/<product>/<id>.<ext>.
func imageIDFromKey(key string) uuid.UUID {
	base := key[strings.LastIndex(key, "/")+1:]
	if dot := strings.LastIndex(base, "."); dot >= 0 {
		base = base[:dot]
	}
	id, err := uuid.Parse(base)
	if err != nil {
		return uuid.New()
	}
	return id
}

// ---------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------

func (s *Service) detail(ctx context.Context, exec platform.DBTX, product *domain.Product) (*domain.Detail, error) {
	variants, err := repository.NewVariantRepository(exec).ListByProduct(ctx, product.ID)
	if err != nil {
		return nil, err
	}
	images, err := repository.NewImageRepository(exec).ListByProduct(ctx, product.ID)
	if err != nil {
		return nil, err
	}
	categoryIDs, err := repository.NewCategoryLinkRepository(exec).ListCategoryIDs(ctx, product.ID)
	if err != nil {
		return nil, err
	}

	d := &domain.Detail{Product: *product, Variants: variants, Images: images, CategoryID: categoryIDs}
	if err := s.enrich(ctx, exec, d); err != nil {
		return nil, err
	}
	return d, nil
}

// enrich stitches the product's attribute set (inherited from its categories),
// its display-spec values, and each variant's chosen options onto the detail.
// Every lookup is a single batched query — no N+1.
func (s *Service) enrich(ctx context.Context, exec platform.DBTX, d *domain.Detail) error {
	attrs, err := attributesrepo.New(exec).ListByCategories(ctx, d.CategoryID)
	if err != nil {
		return err
	}
	if len(attrs) == 0 {
		return nil
	}

	attrIDs := make([]uuid.UUID, len(attrs))
	for i := range attrs {
		attrIDs[i] = attrs[i].ID
	}
	valuesByAttr, err := attributesrepo.New(exec).ValuesByAttributes(ctx, attrIDs)
	if err != nil {
		return err
	}

	attrByID := make(map[uuid.UUID]attrdomain.Attribute, len(attrs))
	valueLabel := make(map[uuid.UUID]string)
	for i := range attrs {
		attrByID[attrs[i].ID] = attrs[i]
		for _, v := range valuesByAttr[attrs[i].ID] {
			valueLabel[v.ID] = v.Value
		}
	}

	// Options = the VARIANT axes with their choices.
	for i := range attrs {
		if attrs[i].Role != attrdomain.RoleVariant {
			continue
		}
		vals := valuesByAttr[attrs[i].ID]
		axis := domain.OptionAxis{
			AttributeID: attrs[i].ID,
			Code:        attrs[i].Code,
			Name:        attrs[i].Name,
			Values:      make([]domain.OptionValue, len(vals)),
		}
		for j, v := range vals {
			axis.Values[j] = domain.OptionValue{ID: v.ID, Value: v.Value}
		}
		d.Options = append(d.Options, axis)
	}

	// Specs = the SPEC attributes that have a stored value for this product.
	specRows, err := repository.NewSpecRepository(exec).ListByProduct(ctx, d.Product.ID)
	if err != nil {
		return err
	}
	for _, row := range specRows {
		a, ok := attrByID[row.AttributeID]
		if !ok {
			continue // attribute removed from the category since
		}
		d.Specs = append(d.Specs, domain.SpecEntry{Code: a.Code, Name: a.Name, Value: row.Value})
	}

	// Each variant's chosen value per axis, keyed by attribute code.
	if len(d.Variants) == 0 {
		return nil
	}
	variantIDs := make([]uuid.UUID, len(d.Variants))
	for i := range d.Variants {
		variantIDs[i] = d.Variants[i].ID
	}
	optRows, err := repository.NewVariantOptionRepository(exec).ByVariants(ctx, variantIDs)
	if err != nil {
		return err
	}
	d.VariantOptions = make(map[uuid.UUID]map[string]string)
	for _, row := range optRows {
		a, ok := attrByID[row.AttributeID]
		if !ok {
			continue
		}
		if d.VariantOptions[row.VariantID] == nil {
			d.VariantOptions[row.VariantID] = make(map[string]string)
		}
		d.VariantOptions[row.VariantID][a.Code] = valueLabel[row.AttributeValueID]
	}
	return nil
}

// resolveVariantOptions parses the request options and checks each names a
// VARIANT axis defined for the product's categories. One query loads that set;
// the value↔attribute pairing is enforced by a composite FK on insert.
func (s *Service) resolveVariantOptions(ctx context.Context, exec platform.DBTX, productID uuid.UUID, in []dto.VariantOptionInput) ([]domain.VariantOption, error) {
	if len(in) == 0 {
		return nil, nil
	}
	catIDs, err := repository.NewCategoryLinkRepository(exec).ListCategoryIDs(ctx, productID)
	if err != nil {
		return nil, err
	}
	attrs, err := attributesrepo.New(exec).ListByCategories(ctx, catIDs)
	if err != nil {
		return nil, err
	}
	isAxis := make(map[uuid.UUID]bool)
	for i := range attrs {
		if attrs[i].Role == attrdomain.RoleVariant {
			isAxis[attrs[i].ID] = true
		}
	}

	out := make([]domain.VariantOption, 0, len(in))
	for _, o := range in {
		attrID, err := uuid.Parse(o.AttributeID)
		if err != nil || !isAxis[attrID] {
			return nil, domain.ErrAttributeNotForProduct
		}
		valueID, err := uuid.Parse(o.ValueID)
		if err != nil {
			return nil, domain.ErrOptionValueInvalid
		}
		out = append(out, domain.VariantOption{AttributeID: attrID, AttributeValueID: valueID})
	}
	return out, nil
}

// SetSpecs replaces a product's whole set of display-spec values.
func (s *Service) SetSpecs(ctx context.Context, tenantID, productID uuid.UUID, req dto.SetSpecsRequest) (*domain.Detail, error) {
	product, err := repository.NewProductRepository(s.db).GetByID(ctx, tenantID, productID)
	if err != nil {
		return nil, err
	}

	err = platform.RunInTx(ctx, s.db, func(tx *sql.Tx) error {
		catIDs, err := repository.NewCategoryLinkRepository(tx).ListCategoryIDs(ctx, productID)
		if err != nil {
			return err
		}
		attrs, err := attributesrepo.New(tx).ListByCategories(ctx, catIDs)
		if err != nil {
			return err
		}
		isSpec := make(map[uuid.UUID]bool)
		for i := range attrs {
			if attrs[i].Role == attrdomain.RoleSpec {
				isSpec[attrs[i].ID] = true
			}
		}

		specs := make([]domain.ProductSpec, 0, len(req.Specs))
		at := make(map[uuid.UUID]int) // attribute id -> index in specs (last value wins)
		for _, in := range req.Specs {
			attrID, err := uuid.Parse(in.AttributeID)
			if err != nil || !isSpec[attrID] {
				return domain.ErrAttributeNotForProduct
			}
			value := strings.TrimSpace(in.Value)
			if idx, dup := at[attrID]; dup {
				specs[idx].Value = value
				continue
			}
			at[attrID] = len(specs)
			specs = append(specs, domain.ProductSpec{ProductID: productID, AttributeID: attrID, Value: value})
		}
		return repository.NewSpecRepository(tx).Replace(ctx, productID, specs)
	})
	if err != nil {
		return nil, err
	}
	return s.detail(ctx, s.db, product)
}

func applyVariantPatch(v *domain.Variant, req dto.UpdateVariantRequest) {
	if req.SKU != nil {
		v.SKU = strings.TrimSpace(*req.SKU)
	}
	if req.Title != nil {
		v.Title = trimPtr(req.Title)
	}
	if req.PriceCents != nil {
		v.PriceCents = *req.PriceCents
	}
	if req.Currency != nil {
		v.Currency = *req.Currency
	}
	if req.CompareAtPriceCents != nil {
		v.CompareAtPriceCents = req.CompareAtPriceCents
	}
	if req.WeightGrams != nil {
		v.WeightGrams = req.WeightGrams
	}
	if req.Barcode != nil {
		v.Barcode = trimPtr(req.Barcode)
	}
	if req.Position != nil {
		v.Position = *req.Position
	}
	if req.IsDefault != nil {
		v.IsDefault = *req.IsDefault
	}
}

func normalizeSlug(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func currencyOrDefault(c *string) string {
	if c == nil || *c == "" {
		return "USD"
	}
	return *c
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}
