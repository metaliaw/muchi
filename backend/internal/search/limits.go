package search

// maxConcurrentSources Bounds how many Stores one Question Asks at once.
const maxConcurrentSources = 4

// maxCheckedOffers Caps one Question. The Caller Asks for the cheapest Offer of
// each Card, not for a whole Page: a longer List would Visit Stores that no
// Recommendation Depends on.
const maxCheckedOffers = 50

// maxCheckoutUnits Caps the Units of one Line. A Deck Asks for four of a Card;
// a Number far above that is a Typo, not a Purchase.
const maxCheckoutUnits = 99

// maxResultPage Reads the Results the same Way the public Route Caps them.
const maxResultPage = 100
