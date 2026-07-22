package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	reportingv1 "sms-reporting/gen/go/reporting/v1"
	"sms-reporting/internal/domain"
	"sms-reporting/internal/service"
)

type ReportingGrpcHandler struct {
	reportingv1.UnimplementedReportingServiceServer
	service service.ReportingService
}

func NewReportingGrpcHandler(service service.ReportingService) *ReportingGrpcHandler {
	return &ReportingGrpcHandler{
		service: service,
	}
}

func (h *ReportingGrpcHandler) RequestReport(ctx context.Context, req *reportingv1.RequestReportRequest) (*reportingv1.RequestReportResponse, error) {
	if req.TargetEmail == "" {
		return nil, status.Error(codes.InvalidArgument, "target_email is required")
	}

	err := h.service.RequestReport(ctx, req.TargetEmail, req.StartDate, req.EndDate)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidEmail) || errors.Is(err, domain.ErrInvalidDateRange) || errors.Is(err, domain.ErrInvalidDateFormat) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "failed to request report: %v", err)
	}

	return &reportingv1.RequestReportResponse{
		Status:  "processing",
		Code:    202,
		Message: "Report request accepted and is being processed in the background",
	}, nil
}
