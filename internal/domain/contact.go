package domain

// CanContactMakers is false for vendors. Guests and artists may open
// the inquiry CTA on a published maker page.
func CanContactMakers(role string) bool {
	return role != "vendor"
}
