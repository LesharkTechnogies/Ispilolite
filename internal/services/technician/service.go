package technician

import("fmt";"strings";"time";"ispilolite/internal/models";"ispilolite/internal/repository";"ispilolite/internal/utils")
type Service struct{repo repository.TechnicianRepository}
func NewService(r repository.TechnicianRepository)*Service{return &Service{r}}
func(s *Service)UpsertProfile(id string,p *models.TechnicianProfile)error{p.TechnicianID=id;p.Bio=strings.TrimSpace(p.Bio);p.County=strings.TrimSpace(p.County);p.Town=strings.TrimSpace(p.Town);p.Village=strings.TrimSpace(p.Village);if p.ExperienceYears<0||p.PricePerHour<0{return fmt.Errorf("experience and price cannot be negative")};return s.repo.UpsertProfile(p)}
func(s *Service)GetProfile(id string)(*models.TechnicianProfile,error){return s.repo.GetProfile(id)}
func(s *Service)Search(skill string,available bool,limit int)([]*models.TechnicianProfile,error){return s.repo.SearchTechnicians(strings.TrimSpace(skill),available,limit)}
func(s *Service)CreatePost(id string,p *models.TechnicianPost)(*models.TechnicianPost,error){p.ID=utils.GenerateID();p.TechnicianID=id;p.Title=strings.TrimSpace(p.Title);p.ServiceType=strings.TrimSpace(p.ServiceType);if p.Title==""||p.ServiceType==""{return nil,fmt.Errorf("title and service_type are required")};if p.Status==""{p.Status="draft"};if p.Status!="draft"&&p.Status!="published"{return nil,fmt.Errorf("invalid post status")};p.CreatedAt=time.Now().UTC();p.UpdatedAt=p.CreatedAt;if err:=s.repo.CreatePost(p);err!=nil{return nil,err};return p,nil}
func(s *Service)UpdatePost(id,owner string,p *models.TechnicianPost)(*models.TechnicianPost,error){p.ID=id;p.TechnicianID=owner;p.Title=strings.TrimSpace(p.Title);p.ServiceType=strings.TrimSpace(p.ServiceType);if p.Status!="draft"&&p.Status!="published"&&p.Status!="archived"{return nil,fmt.Errorf("invalid post status")};if err:=s.repo.UpdatePost(p);err!=nil{return nil,err};return s.repo.GetPostByID(id)}
func(s *Service)Portfolio(id string)([]*models.TechnicianPost,error){items,err:=s.repo.GetPostsByTechnician(id);if err!=nil{return nil,err};out:=items[:0];for _,p:=range items{if p.Status=="published"{out=append(out,p)}};return out,nil}
func(s *Service)MyPosts(id string)([]*models.TechnicianPost,error){return s.repo.GetPostsByTechnician(id)}
func(s *Service)Published(kind string,limit int)([]*models.TechnicianPost,error){return s.repo.ListPublishedPosts(strings.TrimSpace(kind),limit)}
