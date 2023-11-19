package wotlk

// BillingDetails used in ServerAuthResponse.
type BillingDetails struct {
	BillingTimeRemaining uint32
	BillingFlags         uint8
	BillingTimeRested    uint32
}
