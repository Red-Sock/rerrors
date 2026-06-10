package rerrors

import (
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/types/known/durationpb"
)

func appendDetail(e *Error, d protoadapt.MessageV1) {
	*e.grpcDetails = append(*e.grpcDetails, d)
}

func WithBadRequest(field, description string) opt {
	return func(e *Error) {
		appendDetail(e, &errdetails.BadRequest{
			FieldViolations: []*errdetails.BadRequest_FieldViolation{
				{Field: field, Description: description},
			},
		})
	}
}

func WithErrorInfo(reason, domain string, metadata map[string]string) opt {
	return func(e *Error) {
		appendDetail(e, &errdetails.ErrorInfo{
			Reason:   reason,
			Domain:   domain,
			Metadata: metadata,
		})
	}
}

func WithRetryInfo(delay time.Duration) opt {
	return func(e *Error) {
		appendDetail(e, &errdetails.RetryInfo{
			RetryDelay: durationpb.New(delay),
		})
	}
}

func WithDebugInfo(detail string, stackEntries ...string) opt {
	return func(e *Error) {
		appendDetail(e, &errdetails.DebugInfo{
			Detail:       detail,
			StackEntries: stackEntries,
		})
	}
}

func WithQuotaFailure(subject, description string) opt {
	return func(e *Error) {
		appendDetail(e, &errdetails.QuotaFailure{
			Violations: []*errdetails.QuotaFailure_Violation{
				{Subject: subject, Description: description},
			},
		})
	}
}

func WithPreconditionFailure(type_, subject, description string) opt {
	return func(e *Error) {
		appendDetail(e, &errdetails.PreconditionFailure{
			Violations: []*errdetails.PreconditionFailure_Violation{
				{Type: type_, Subject: subject, Description: description},
			},
		})
	}
}

func WithRequestInfo(requestID, servingData string) opt {
	return func(e *Error) {
		appendDetail(e, &errdetails.RequestInfo{
			RequestId:   requestID,
			ServingData: servingData,
		})
	}
}

func WithResourceInfo(resourceType, resourceName, owner, description string) opt {
	return func(e *Error) {
		appendDetail(e, &errdetails.ResourceInfo{
			ResourceType: resourceType,
			ResourceName: resourceName,
			Owner:        owner,
			Description:  description,
		})
	}
}

func WithHelp(url, description string) opt {
	return func(e *Error) {
		appendDetail(e, &errdetails.Help{
			Links: []*errdetails.Help_Link{
				{Url: url, Description: description},
			},
		})
	}
}

func WithLocalizedMessage(locale, message string) opt {
	return func(e *Error) {
		appendDetail(e, &errdetails.LocalizedMessage{
			Locale:  locale,
			Message: message,
		})
	}
}
