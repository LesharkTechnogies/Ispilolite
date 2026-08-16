package isp

import (
	"fmt"
	"ispilolite/internal/models"
	"ispilolite/internal/repository"
	"strings"
	"time"
)

type ISPService struct {
	ispRepo repository.ISPRepository
}

func NewISPService(ispRepo repository.ISPRepository) *ISPService {
	return &ISPService{
		ispRepo: ispRepo,
	}
}

func (s *ISPService) GetISPs() ([]*models.ISP, error) {
	return s.ispRepo.GetISPs()
}
func (s *ISPService) ListISPsByPackage(filter models.PackageFilter)([]*models.ISP,error){filter.Category=strings.ToLower(strings.TrimSpace(filter.Category));filter.Sort=strings.ToLower(strings.TrimSpace(filter.Sort));return s.ispRepo.ListISPsByPackage(filter)}

func (s *ISPService) GetISPByID(ispID string) (*models.ISP, error) {
	return s.ispRepo.GetISPByID(ispID)
}

func (s *ISPService) GetISPPackages(ispID string) ([]*models.ISPPackage, error) {
	return s.ispRepo.GetISPPackages(ispID)
}

func (s *ISPService) UpdateISP(isp *models.ISP) error { return s.ispRepo.UpdateISP(isp) }
func (s *ISPService) CreatePackage(pkg *models.ISPPackage) error { if err:=validatePackage(pkg);err!=nil{return err};if err:=s.ispRepo.ValidatePackageUnits(pkg.SpeedUnitID,pkg.CapacityUnitID,pkg.CapacityType);err!=nil{return err};return s.ispRepo.CreatePackage(pkg) }
func (s *ISPService) UpdatePackage(pkg *models.ISPPackage) error { if err:=validatePackage(pkg);err!=nil{return err};if err:=s.ispRepo.ValidatePackageUnits(pkg.SpeedUnitID,pkg.CapacityUnitID,pkg.CapacityType);err!=nil{return err};return s.ispRepo.UpdatePackage(pkg) }
func (s *ISPService) ListPackages(filter models.PackageFilter)([]*models.ISPPackage,error){filter.Category=strings.ToLower(strings.TrimSpace(filter.Category));filter.Sort=strings.ToLower(strings.TrimSpace(filter.Sort));if filter.MinPrice<0||filter.MaxPrice<0||filter.MinSpeed<0||filter.MaxSpeed<0{return nil,fmt.Errorf("package filters cannot be negative")};if filter.MaxPrice>0&&filter.MinPrice>filter.MaxPrice{return nil,fmt.Errorf("min_price cannot exceed max_price")};if filter.MaxSpeed>0&&filter.MinSpeed>filter.MaxSpeed{return nil,fmt.Errorf("min_speed cannot exceed max_speed")};return s.ispRepo.ListPackages(filter)}
func (s *ISPService) SetPackageCountyPrice(packageID,ispID,county string,price float64)error{if strings.TrimSpace(county)==""||price<0{return fmt.Errorf("county and non-negative price are required")};return s.ispRepo.SetPackageCountyPrice(packageID,ispID,county,price)}
func (s *ISPService) ListPackageUnits(dimension string)([]*models.PackageUnit,error){dimension=strings.ToLower(strings.TrimSpace(dimension));if dimension!=""&&dimension!="bandwidth"&&dimension!="data"{return nil,fmt.Errorf("dimension must be bandwidth or data")};return s.ispRepo.ListPackageUnits(dimension)}
func(s *ISPService)ArchivePackage(id,ispID string)error{return s.ispRepo.ArchivePackage(id,ispID)}
func(s *ISPService)DeletePackage(id,ispID string)error{return s.ispRepo.DeletePackage(id,ispID)}
func(s *ISPService)ReservePackage(id,customerID,county string)(string,error){return s.ispRepo.ReservePackage(id,customerID,strings.TrimSpace(county),time.Now().UTC().Add(15*time.Minute))}
func(s *ISPService)Subscribe(reservationID,customerID string)(*models.PackageSubscription,error){return s.ispRepo.CreatePackageSubscription(reservationID,customerID)}
func(s *ISPService)UpdateSubscription(id,actor,status string,endsAt *time.Time)error{status=strings.ToLower(strings.TrimSpace(status));valid:=map[string]bool{"active":true,"suspended":true,"cancelled":true,"expired":true};if !valid[status]{return fmt.Errorf("invalid subscription status")};return s.ispRepo.UpdatePackageSubscription(id,actor,status,endsAt)}
func(s *ISPService)ListSubscriptions(id,role,status string,limit int)([]*models.PackageSubscription,error){if limit<=0||limit>100{limit=50};return s.ispRepo.ListPackageSubscriptions(id,role,strings.ToLower(strings.TrimSpace(status)),limit)}
func validatePackage(pkg *models.ISPPackage)error{pkg.Name=strings.TrimSpace(pkg.Name);pkg.Category=strings.ToLower(strings.TrimSpace(pkg.Category));pkg.BillingCycle=strings.ToLower(strings.TrimSpace(pkg.BillingCycle));pkg.CapacityType=strings.ToLower(strings.TrimSpace(pkg.CapacityType));if pkg.Name==""||pkg.SpeedValue<=0||pkg.SpeedUnitID==""||pkg.BasePrice<0{return fmt.Errorf("name, positive speed, speed unit, and non-negative price are required")};if pkg.Category!="pppoe"&&pkg.Category!="hotspot"{return fmt.Errorf("category must be pppoe or hotspot")};if pkg.BillingCycle==""{pkg.BillingCycle="monthly"};if pkg.CapacityType==""{pkg.CapacityType="unlimited"};if pkg.CapacityType!="unlimited"&&pkg.CapacityType!="capped"{return fmt.Errorf("capacity_type must be unlimited or capped")};if pkg.CapacityType=="capped"&&(pkg.CapacityValue<=0||pkg.CapacityUnitID==""){return fmt.Errorf("capped packages require capacity_value and capacity_unit_id")};if pkg.CapacityType=="unlimited"{pkg.CapacityValue=0;pkg.CapacityUnitID=""};return nil}
