# Todo List

This file lists all the functions that need to be implemented and where they should be located.

## `api/handlers/auth_handler.go`

- [x] `GetMyProfile` (for clients)
- [x] `UpdateMyProfile` (for clients)
- [x] `GetMyISPProfile` (for ISPs)
- [x] `UpdateMyISPProfile` (for ISPs)
- [x] `GetMyTechProfile` (for technicians)

## `api/handlers/job_handler.go`

- [x] `GetISPs`
- [x] `GetISPByID`
- [ ] `GetISPPackages`
- [ ] `GetISPReviews`
- [ ] `CreateInstallation`
- [ ] `GetMyInstallations`
- [ ] `CreateISPReview`
- [ ] `GetMyISPInstallations`
- [ ] `UpdateInstallation`
- [ ] `GetMyTechnicians`
- [ ] `CreateTechnician`
- [ ] `DeleteTechnician`
- [ ] `CreatePackage`
- [ ] `UpdatePackage`
- [ ] `GetMyTechJobs`
- [ ] `UpdateJobStatus`

## `api/routes/routes.go`

- [ ] The routes for `/search/locations`, `/search/isps`, and `/search/technicians` are currently using a `placeholderHandler`. These need to be wired up to use the existing `SearchHandler` methods in `api/handlers/search_handler.go`.

## Empty Files to be Implemented

The following files are empty and need to have their respective functionalities implemented.

### Handlers
- [ ] `api/handlers/auth_handler.go`
- [x] `api/handlers/job_handler.go`
- [ ] `api/handlers/location_handler.go`

### Middleware
- [ ] `internal/middleware/logger.go`
- [ ] `internal/middleware/ratelimit.go`

### Models
- [ ] `internal/models/job.go`
- [ ] `internal/models/quotation.go`

### Repositories
- [ ] `internal/repository/elasticsearch/search_repo.go`
- [ ] `internal/repository/postgres/job_repo.go`
- [ ] `internal/repository/postgres/location_repo.go`
- [ ] `internal/repository/postgres/user_repo.go`
- [ ] `internal/repository/redis/cache.go`
- [ ] `internal/repository/redis/session.go`

### Services
- [ ] `internal/services/auth/auth_service.go`
- [ ] `internal/services/geospatial/geospatial_service.go`
- [ ] `internal/services/matching/matching_service.go`
- [ ] `internal/services/notification/notification_service.go`

### Utils
- [ ] `internal/utils/geoutils.go`
- [ ] `internal/utils/watermark.go`
