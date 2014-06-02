package crud

var (
	EnableUpsert      bool
	NoLastInsertId    bool
	PlaceholderFormat func(id int) string = func(id int) string {
		return "?"
	}
)
