package types

// AlaskaResponse is the JSON response from Alaska's searchbff/V3/search endpoint.
// Reverse-engineered from AwardWiz (github.com/lg/awardwiz).
type AlaskaResponse struct {
	DepartureStation   string           `json:"departureStation"`
	ArrivalStation     string           `json:"arrivalStation"`
	Slices             []AlaskaSlice    `json:"slices,omitempty"`
	Env                string           `json:"env"`
	QPXCSessionID      string           `json:"qpxcSessionID"`
	QPXCSolutionSetID  string           `json:"qpxcSolutionSetID"`
	Advisories         []any            `json:"advisories"`
}

type AlaskaSlice struct {
	ID          int                       `json:"id"`
	Origin      string                    `json:"origin"`
	Destination string                    `json:"destination"`
	Duration    int                       `json:"duration"`
	Segments    []AlaskaSegment           `json:"segments"`
	Fares       map[string]AlaskaFare     `json:"fares"`
	UpgradeInfo []any                     `json:"upgradeInfo"`
}

type AlaskaSegment struct {
	PublishingCarrier AlaskaCarrier   `json:"publishingCarrier"`
	DisplayCarrier   AlaskaCarrier   `json:"displayCarrier"`
	DepartureStation string          `json:"departureStation"`
	ArrivalStation   string          `json:"arrivalStation"`
	AircraftCode     string          `json:"aircraftCode"`
	Aircraft         string          `json:"aircraft"`
	Duration         int             `json:"duration"`
	DepartureTime    string          `json:"departureTime"`
	ArrivalTime      string          `json:"arrivalTime"`
	NextDayArrival   bool            `json:"nextDayArrival"`
	NextDayDeparture bool            `json:"nextDayDeparture"`
	Amenities        []string        `json:"amenities"`
	FirstAmenities   []string        `json:"firstAmenities"`
	FirstClassUpgradeAvailable   bool `json:"firstClassUpgradeAvailable"`
	FirstClassUpgradeUnavailable bool `json:"firstClassUpgradeUnavailable"`
}

type AlaskaCarrier struct {
	CarrierCode     string `json:"carrierCode"`
	CarrierFullName string `json:"carrierFullName"`
	FlightNumber    int    `json:"flightNumber"`
}

type AlaskaFare struct {
	GrandTotal      float64  `json:"grandTotal"`
	MilesPoints     int      `json:"milesPoints"`
	SeatsRemaining  int      `json:"seatsRemaining"`
	Discount        bool     `json:"discount"`
	MixedCabin      bool     `json:"mixedCabin"`
	Cabins          []string `json:"cabins"`
	BookingCodes    []string `json:"bookingCodes"`
	Refundable      bool     `json:"refundable"`
	QPXCSolutionID  string   `json:"qpxcSolutionID"`
}

// NormalizedFlight is our clean output format.
type NormalizedFlight struct {
	FlightNumber string           `json:"flightNumber"`
	Carrier      CarrierInfo      `json:"carrier"`
	Departure    AirportTime      `json:"departure"`
	Arrival      AirportTime      `json:"arrival"`
	Duration     int              `json:"duration"` // minutes
	Aircraft     string           `json:"aircraft"`
	IsDirect     bool             `json:"isDirect"`
	Fares        []NormalizedFare `json:"fares"`
	Amenities    []string         `json:"amenities"`
}

type CarrierInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type AirportTime struct {
	Airport string `json:"airport"`
	Time    string `json:"time"` // ISO 8601
}

type NormalizedFare struct {
	Cabin          string  `json:"cabin"` // economy, business, first
	Miles          int     `json:"miles"`
	Cash           float64 `json:"cash"`
	SeatsRemaining int     `json:"seatsRemaining"`
	BookingCode    string  `json:"bookingCode"`
	IsSaver        bool    `json:"isSaver"`
}

// CabinName maps Alaska's cabin designations to standard names.
func CabinName(alaskaCabin string) string {
	switch alaskaCabin {
	case "FIRST":
		return "business" // Alaska "First" on domestic = business class
	case "BUSINESS":
		return "business"
	case "MAIN", "COACH", "SAVER":
		return "economy"
	default:
		return "economy"
	}
}

// IsSaverCabin returns true if the cabin designation is a saver fare.
func IsSaverCabin(cabin string) bool {
	return cabin == "SAVER"
}
