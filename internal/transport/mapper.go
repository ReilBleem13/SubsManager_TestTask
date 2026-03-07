package transport

import "github.com/ReilBleem13/internal/service"

func mapCreateSubJSONToService(in *CreateSubJSON) *service.CreateSubRequest {
	return &service.CreateSubRequest{
		ServiceName: in.ServiceName,
		Price:       in.Price,
		UserID:      in.UserID,
		StartDate:   in.StartDate,
		EndData:     in.EndData,
	}
}

func mapUpdateSubJSONToService(in *UpdateSubJSON) *service.UpdateSubRequest {
	return &service.UpdateSubRequest{
		ServiceName: in.ServiceName,
		Price:       in.Price,
		UserID:      in.UserID,
		StartDate:   in.StartDate,
		EndDate:     in.EndData,
	}
}
