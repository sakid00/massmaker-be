package domain

import "github.com/google/uuid"

const (
	ConsentActivityTracking          = "activity_tracking"
	ConsentMarketing                 = "marketing"
	ConsentWhatsAppPublication       = "whatsapp_publication"
	ConsentRecommendationPublication = "recommendation_publication"
	ConsentPrivacyPolicy             = "privacy_policy"
	ConsentUserAgreement             = "user_agreement"
	ConsentTerms                     = "terms"

	ConsentSourceSignup         = "signup"
	ConsentSourceOnboarding     = "onboarding"
	ConsentSourceActivityBanner = "activity_banner"
	ConsentSourceFooter         = "footer"
)

func GuestConsentType(t string) bool {
	return t == ConsentActivityTracking || t == ConsentMarketing
}

func DocumentConsentType(t string) bool {
	switch t {
	case ConsentPrivacyPolicy, ConsentUserAgreement, ConsentTerms:
		return true
	default:
		return false
	}
}

func ConsentTypeAllowed(role, consentType string) bool {
	switch consentType {
	case ConsentActivityTracking, ConsentMarketing:
		return true
	case ConsentPrivacyPolicy, ConsentUserAgreement, ConsentTerms:
		return role != ""
	case ConsentWhatsAppPublication:
		return role == "vendor"
	case ConsentRecommendationPublication:
		return role == "artist"
	default:
		return false
	}
}

func ConsentSourceAllowed(source string) bool {
	switch source {
	case ConsentSourceSignup, ConsentSourceOnboarding, ConsentSourceActivityBanner, ConsentSourceFooter:
		return true
	default:
		return false
	}
}

// ShowPublicWhatsApp is true only with staff permit, and — once a vendor
// account is bound — that vendor's current whatsapp_publication grant.
func ShowPublicWhatsApp(staffPermitted bool, vendorUserID *uuid.UUID, vendorGranted bool) bool {
	if !staffPermitted {
		return false
	}
	if vendorUserID == nil {
		return true
	}
	return vendorGranted
}
