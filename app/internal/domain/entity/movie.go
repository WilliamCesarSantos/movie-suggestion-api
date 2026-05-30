package entity

type Genre struct{ Name string }
type Actor struct {
	Name string
}
type Director struct {
	Name string
}

type Movie struct {
	ID         string
	Title      string
	Year       string
	Plot       string
	Runtime    string
	Poster     string
	ImdbRating float64
	ImdbID     string
	Genres     []Genre
	Actors     []Actor
	Directors  []Director
}
