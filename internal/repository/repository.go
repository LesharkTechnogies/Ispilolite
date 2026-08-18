package repository

import (
	"ispilolite/internal/models"
	"time"
)

type UserRepository interface {
	CreateUser(user *models.User) error
	GetUserByPhone(phone string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(userID string) (*models.User, error)
	UpdateUser(user *models.User) error
	GetTechniciansByISPID(ispID string) ([]*models.User, error)
	GetUsersByStatus(status string) ([]*models.User, error)
	ListUsersForMessaging(role string, userIDs []string) ([]*models.User, error)
	RequestDeleteUser(userID string, status string) error
	SanitizeAndDeleteUser(userID string) error
	CreateRefreshSession(sessionID, userID, tokenHash string, expiresAt time.Time) error
	RefreshSessionActive(sessionID, tokenHash string) (bool, error)
}

type ISPRepository interface {
	GetISPs() ([]*models.ISP, error)
	ListISPsByPackage(filter models.PackageFilter) ([]*models.ISP, error)
	GetISPByID(ispID string) (*models.ISP, error)
	GetISPPackages(ispID string) ([]*models.ISPPackage, error)
	CreateISP(isp *models.ISP) error
	UpdateISP(isp *models.ISP) error
	CreatePackage(pkg *models.ISPPackage) error
	UpdatePackage(pkg *models.ISPPackage) error
	ListPackages(filter models.PackageFilter) ([]*models.ISPPackage, error)
	SetPackageCountyPrice(packageID, ispID, county string, price float64) error
	ListPackageUnits(dimension string) ([]*models.PackageUnit, error)
	ValidatePackageUnits(speedUnitID, capacityUnitID, capacityType string) error
	ArchivePackage(packageID, ispID string) error
	DeletePackage(packageID, ispID string) error
	ReservePackage(packageID, customerID, county string, expiresAt time.Time) (string, error)
	CreatePackageSubscription(reservationID, customerID string) (*models.PackageSubscription, error)
	UpdatePackageSubscription(subscriptionID, actorID, status string, endsAt *time.Time) error
	ListPackageSubscriptions(userID, role, status string, limit int) ([]*models.PackageSubscription, error)
}

type ReviewRepository interface {
	CreateReview(review *models.Review) error
	GetReviewsByTarget(targetID string, targetType string) ([]*models.Review, error)
	AnonymizeReviewsByUserID(userID string) error
	ReportReview(report *models.ReviewReport) error
	ModerateReview(reviewID, adminID, status, note string) error
	ListPendingReviews(limit int) ([]*models.Review, error)
}

type FlagRepository interface {
	CreateFlag(flag *models.Flag) error
}

type InstallationRepository interface {
	CreateInstallation(installation *models.Installation) error
	GetInstallationsByClientID(clientID string) ([]*models.Installation, error)
	GetInstallationsByISPID(ispID string) ([]*models.Installation, error)
	GetInstallationByID(installationID string) (*models.Installation, error)
	UpdateInstallation(installation *models.Installation) error
}

type TechnicianRepository interface {
	UpsertProfile(profile *models.TechnicianProfile) error
	GetProfile(technicianID string) (*models.TechnicianProfile, error)
	SetSkills(technicianID string, skills []string) error
	GetSkills(technicianID string) ([]string, error)
	SearchTechnicians(skill string, availableOnly bool, limit int) ([]*models.TechnicianProfile, error)
	CreatePost(post *models.TechnicianPost) error
	UpdatePost(post *models.TechnicianPost) error
	GetPostByID(postID string) (*models.TechnicianPost, error)
	GetPostsByTechnician(technicianID string) ([]*models.TechnicianPost, error)
	ListPublishedPosts(serviceType string, limit int) ([]*models.TechnicianPost, error)
}

type JobRequestRepository interface {
	CreateJobRequest(req *models.JobRequest) error
	GetJobRequestByID(id string) (*models.JobRequest, error)
	ListForCustomer(customerID string, status string) ([]*models.JobRequest, error)
	ListForTechnician(technicianID string, status string) ([]*models.JobRequest, error)
	ListAvailable(county, town, serviceType string) ([]*models.JobRequest, error)
	ApplyToJob(application *models.JobApplication) error
	ListApplications(requestID, customerID string) ([]*models.JobApplication, error)
	AssignApplication(requestID, customerID, applicationID string) (*models.JobRequest, error)
	SetAvailability(requestID, customerID string, available bool) error
	DeleteJobRequest(requestID, customerID string) error
	UpdateJobStatus(requestID, actorID, status string) error
}

type LocationRepository interface {
	// CreateLocation inserts a new (typically pending) place.
	CreateLocation(location *models.Location) error
	// GetLocationByID returns a place by id, or (nil, nil) if not found.
	GetLocationByID(id string) (*models.Location, error)
	// FindLocationByName looks up a place by its dedup key (case-insensitive
	// name + type + parent). Returns (nil, nil) when no match exists. This is
	// what lets automatic town-adding reuse an existing row instead of
	// creating a duplicate.
	FindLocationByName(name, locationType, parentID string) (*models.Location, error)
	// SearchLocations returns places matching a name prefix/substring, most
	// popular first. locationType filters by county/town/village when set.
	SearchLocations(query, locationType string, limit int) ([]*models.Location, error)
	// RecordSubmission stores one user's submission of a place. The bool result
	// reports whether it was a new distinct submission (false when the same
	// user already submitted this place).
	RecordSubmission(submission *models.LocationSubmission) (bool, error)
	// CountSubmissions returns the number of distinct submissions for a place.
	CountSubmissions(locationID string) (int, error)
	// UpdateLocationStats persists a recomputed submission count, popularity,
	// verification flag and status after a new submission.
	UpdateLocationStats(id string, submissionCount int, popularityScore float64, isVerified bool, status string) error
	ListLocationsByCounty(county string, limit int) ([]*models.Location, error)
	CreateAlias(alias *models.LocationAlias) error
	SearchAliases(query string, limit int) ([]*models.Location, error)
	ValidateBoundary(locationID string, lat, lon float64) (bool, error)
}

type CoverageRepository interface {
	ListISPCoverage(ispID, county string) ([]*models.Location, error)
	AddISPCoverage(ispID, locationID string) error
	ListCoverageRecommendations(ispID, county string, limit int) ([]*models.Location, error)
}

type NotificationRepository interface {
	CreateNotification(notification *models.Notification) error
	ListNotifications(userID string, unreadOnly bool, limit int) ([]*models.Notification, error)
	MarkNotificationRead(userID, notificationID string) error
}

type QuotationRepository interface {
	NextQuotationNumber(at time.Time) (string, error)
	PublicCodeExists(code string) (bool, error)
	FinalizeQuotation(quotation *models.Quotation) error
	GetQuotationByID(id string) (*models.Quotation, error)
	GetQuotationByPublicCode(code string) (*models.Quotation, error)
	ListQuotations(userID, role, status string, limit int) ([]*models.Quotation, error)
	UpdateQuotationStatus(id, customerID, status string) error
	GetUnit(unitID, issuerID string) (*models.QuotationUnit, error)
	ListUnits(issuerID, query string, limit int) ([]*models.QuotationUnit, error)
	CreateUnit(unit *models.QuotationUnit, issuerID string) error
	GetTaxRate(id string) (*models.TaxRate, error)
	CanQuoteRequest(requestID, issuerID, customerID string) (bool, error)
	GetDocumentForUser(documentID, userID string, public bool) (*models.Document, error)
	GetDocumentForQuotation(quotationID, userID string) (*models.Document, error)
	GetDocumentByQuotation(quotationID string) (*models.Document, error)
	CreateQuotationDocument(document *models.Document) error
}

type CacheRepository interface {
	SetOTP(userID string, otp string, expiration time.Duration) error
	GetOTP(userID string) (string, error)
	DeleteOTP(userID string) error
	Set(key string, value interface{}, expiration time.Duration) error
	Get(key string) (string, error)
	SetRevokedToken(token string, expiration time.Duration) error
	IsTokenRevoked(token string) bool
}
