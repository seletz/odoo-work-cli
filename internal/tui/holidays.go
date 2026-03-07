package tui

import (
	"time"

	"github.com/wlbr/feiertage"
)

// regionFunc maps Bundesland names to feiertage region functions.
var regionFunc = map[string]func(int, ...bool) feiertage.Region{
	"BadenWürttemberg":       feiertage.BadenWürttemberg,
	"Baden-Württemberg":      feiertage.BadenWürttemberg,
	"Bayern":                 feiertage.Bayern,
	"Berlin":                 feiertage.Berlin,
	"Brandenburg":            feiertage.Brandenburg,
	"Bremen":                 feiertage.Bremen,
	"Hamburg":                feiertage.Hamburg,
	"Hessen":                 feiertage.Hessen,
	"MecklenburgVorpommern":  feiertage.MecklenburgVorpommern,
	"Mecklenburg-Vorpommern": feiertage.MecklenburgVorpommern,
	"Niedersachsen":          feiertage.Niedersachsen,
	"NordrheinWestfalen":     feiertage.NordrheinWestfalen,
	"Nordrhein-Westfalen":    feiertage.NordrheinWestfalen,
	"RheinlandPfalz":         feiertage.RheinlandPfalz,
	"Rheinland-Pfalz":        feiertage.RheinlandPfalz,
	"Saarland":               feiertage.Saarland,
	"Sachsen":                feiertage.Sachsen,
	"SachsenAnhalt":          feiertage.SachsenAnhalt,
	"Sachsen-Anhalt":         feiertage.SachsenAnhalt,
	"SchleswigHolstein":      feiertage.SchleswigHolstein,
	"Schleswig-Holstein":     feiertage.SchleswigHolstein,
	"Thüringen":              feiertage.Thüringen,
	"Deutschland":            feiertage.Deutschland,
}

// HolidayMap maps date strings ("2006-01-02") to holiday names.
type HolidayMap map[string]string

// BuildHolidayMap returns a map of holidays for the given year and Bundesland.
// If bundesland is empty or unknown, returns national holidays (Deutschland).
func BuildHolidayMap(year int, bundesland string) HolidayMap {
	fn, ok := regionFunc[bundesland]
	if !ok {
		fn = feiertage.Deutschland
	}
	region := fn(year)

	m := make(HolidayMap, len(region.Feiertage))
	for _, f := range region.Feiertage {
		key := f.Format("2006-01-02")
		m[key] = f.Text
	}
	return m
}

// WeekHolidays returns holiday names for each day (Mon=0..Sun=6) of the
// week starting at monday. Empty string means no holiday.
func WeekHolidays(monday time.Time, holidays HolidayMap) [7]string {
	var result [7]string
	for d := 0; d < 7; d++ {
		day := monday.AddDate(0, 0, d).Format("2006-01-02")
		result[d] = holidays[day]
	}
	return result
}
