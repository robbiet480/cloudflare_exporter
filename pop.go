package main

import (
	"encoding/json"
	"sort"
	"strings"
)

type pop struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Region string `json:"region"`
	Source string `json:"source"`
}

type byName []pop

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name < a[j].Name }

// When this was last generated from cloudflarestatus.com, SJC-PIG and SFO didn't exist on the site and had to be manually added.
const popsJSON = `[{"name":"Auckland, New Zealand","code":"AKL","region":"Oceania"},{"name":"Amsterdam, Netherlands","code":"AMS","region":"Europe"},{"name":"Stockholm, Sweden","code":"ARN","region":"Europe"},{"name":"Athens, Greece","code":"ATH","region":"Europe"},{"name":"Atlanta, GA, United States","code":"ATL","region":"North America"},{"name":"Barcelona, Spain","code":"BCN","region":"Europe"},{"name":"Belgrade, Serbia","code":"BEG","region":"Europe"},{"name":"Beirut, Lebanon","code":"BEY","region":"Middle East"},{"name":"Bangkok, Thailand","code":"BKK","region":"Asia"},{"name":"Nashville, TN, United States","code":"BNA","region":"North America"},{"name":"Brisbane, QLD, Australia","code":"BNE","region":"Oceania"},{"name":"Mumbai, India","code":"BOM","region":"Asia"},{"name":"Boston, MA, United States","code":"BOS","region":"North America"},{"name":"Brussels, Belgium","code":"BRU","region":"Europe"},{"name":"Budapest, HU","code":"BUD","region":"Europe"},{"name":"Cairo, Egypt","code":"CAI","region":"Africa"},{"name":"Guangzhou, China","code":"CAN","region":"Asia"},{"name":"Paris, France","code":"CDG","region":"Europe"},{"name":"Zhengzhou, China","code":"CGO","region":"Asia"},{"name":"Popmbo, Sri Lanka","code":"CMB","region":"Asia"},{"name":"Copenhagen, Denmark","code":"CPH","region":"Europe"},{"name":"Cape Town, South Africa","code":"CPT","region":"Africa"},{"name":"Zuzhou, China","code":"CSX","region":"Asia"},{"name":"Chengdu, China","code":"CTU","region":"Asia"},{"name":"Willemstad, Curaçao","code":"CUR","region":"Latin America & the Caribbean"},{"name":"New Delhi, India","code":"DEL","region":"Asia"},{"name":"Denver, CO, United States","code":"DEN","region":"North America"},{"name":"Dallas, TX, United States","code":"DFW","region":"North America"},{"name":"Moscow, Russia","code":"DME","region":"Europe"},{"name":"Doha, Qatar","code":"DOH","region":"Middle East"},{"name":"Detroit, MI, United States","code":"DTW","region":"North America"},{"name":"Dublin, Ireland","code":"DUB","region":"Europe"},{"name":"Düsseldorf, Germany","code":"DUS","region":"Europe"},{"name":"Dubai, United Arab Emirates","code":"DXB","region":"Middle East"},{"name":"Yerevan, Armenia","code":"EVN","region":"Asia"},{"name":"Newark, NJ, United States","code":"EWR","region":"North America"},{"name":"Buenos Aires, Argentina","code":"EZE","region":"Latin America & the Caribbean"},{"name":"Rome, Italy","code":"FCO","region":"Europe"},{"name":"Fuzhou, China","code":"FOC","region":"Asia"},{"name":"Frankfurt, Germany","code":"FRA","region":"Europe"},{"name":"Foshan, China","code":"FUO","region":"Asia"},{"name":"Rio de Janeiro, Brazil","code":"GIG","region":"Latin America & the Caribbean"},{"name":"São Paulo, Brazil","code":"GRU","region":"Latin America & the Caribbean"},{"name":"Hamburg, Germany","code":"HAM","region":"Europe"},{"name":"Helsinki, Finland","code":"HEL","region":"Europe"},{"name":"Hangzhou, China","code":"HGH","region":"Asia"},{"name":"Hong Kong, Hong Kong","code":"HKG","region":"Asia"},{"name":"Hengyang, China","code":"HNY","region":"Asia"},{"name":"Ashburn, VA, United States","code":"IAD","region":"North America"},{"name":"Seoul, South Korea","code":"ICN","region":"Asia"},{"name":"Indianapolis, IN, United States","code":"IND","region":"North America"},{"name":"Djibouti City, Djibouti","code":"JIB","region":"Africa"},{"name":"Johannesburg, South Africa","code":"JNB","region":"Africa"},{"name":"Kiev, Ukraine","code":"KBP","region":"Europe"},{"name":"Osaka, Japan","code":"KIX","region":"Asia"},{"name":"Kathmandu, Nepal","code":"KTM","region":"Asia"},{"name":"Kuala Lumpur, Malaysia","code":"KUL","region":"Asia"},{"name":"Kuwait City, Kuwait","code":"KWI","region":"Middle East"},{"name":"Luanda, Angola","code":"LAD","region":"Africa"},{"name":"Las Vegas, NV, United States","code":"LAS","region":"North America"},{"name":"Los Angeles, CA, United States","code":"LAX","region":"North America"},{"name":"London, United Kingdom","code":"LHR","region":"Europe"},{"name":"Lima, Peru","code":"LIM","region":"Latin America & the Caribbean"},{"name":"Lisbon, Portugal","code":"LIS","region":"Europe"},{"name":"Luoyang, China","code":"LYA","region":"Asia"},{"name":"Chennai, India","code":"MAA","region":"Asia"},{"name":"Madrid, Spain","code":"MAD","region":"Europe"},{"name":"Manchester, United Kingdom","code":"MAN","region":"Europe"},{"name":"Mombasa, Kenya","code":"MBA","region":"Africa"},{"name":"Kansas City, MO, United States","code":"MCI","region":"North America"},{"name":"Muscat, Oman","code":"MCT","region":"Middle East"},{"name":"Medellín, Columbia","code":"MDE","region":"Latin America & the Caribbean"},{"name":"Melbourne, VIC, Australia","code":"MEL","region":"Oceania"},{"name":"McAllen, TX, United States","code":"MFE","region":"North America"},{"name":"Miami, FL, United States","code":"MIA","region":"North America"},{"name":"Manila, Philippines","code":"MNL","region":"Asia"},{"name":"Marseille, France","code":"MRS","region":"Europe"},{"name":"Port Louis, Mauritius","code":"MRU","region":"Africa"},{"name":"Minneapolis, MN, United States","code":"MSP","region":"North America"},{"name":"Munich, Germany","code":"MUC","region":"Europe"},{"name":"Milan, Italy","code":"MXP","region":"Europe"},{"name":"Langfang, China","code":"NAY","region":"Asia"},{"name":"Nanning, China","code":"NNG","region":"Asia"},{"name":"Tokyo, Japan","code":"NRT","region":"Asia"},{"name":"Omaha, NE, United States","code":"OMA","region":"North America"},{"name":"Chicago, IL, United States","code":"ORD","region":"North America"},{"name":"Oslo, Norway","code":"OSL","region":"Europe"},{"name":"Bucharest, Romania","code":"OTP","region":"Europe"},{"name":"Portland, OR, United States","code":"PDX","region":"North America"},{"name":"Perth, WA, Australia","code":"PER","region":"Oceania"},{"name":"Phoenix, AZ, United States","code":"PHX","region":"North America"},{"name":"Pittsburgh, PA, United States","code":"PIT","region":"North America"},{"name":"Phnom Penh, Cambodia","code":"PNH","region":"Asia"},{"name":"Prague, Czech Republic","code":"PRG","region":"Europe"},{"name":"Panama City, Panama","code":"PTY","region":"Latin America & the Caribbean"},{"name":"San Diego, CA, United States","code":"SAN","region":"North America"},{"name":"Valparaíso, Chile","code":"SCL","region":"Latin America & the Caribbean"},{"name":"Seattle, WA, United States","code":"SEA","region":"North America"},{"name":"San Francisco, CA, United States","code":"SFO","region":"North America"},{"name":"Shenyang, China","code":"SHE","region":"Asia"},{"name":"Singapore, Singapore","code":"SIN","region":"Asia"},{"name":"San Jose, CA, United States","code":"SJC","region":"North America"},{"name":"San Jose (Alternate), CA, United States","code":"SJC-PIG","region":"North America"},{"name":"Shijiazhuang, China","code":"SJW","region":"Asia"},{"name":"Salt Lake City, UT, United States","code":"SLC","region":"North America"},{"name":"Sofia, Bulgaria","code":"SOF","region":"Europe"},{"name":"St. Louis, MO, United States","code":"STL","region":"North America"},{"name":"Sydney, NSW, Australia","code":"SYD","region":"Oceania"},{"name":"Suzhou, China","code":"SZV","region":"Asia"},{"name":"Dongguan, China","code":"SZX","region":"Asia"},{"name":"Qingdao, China","code":"TAO","region":"Asia"},{"name":"Jinan, China","code":"TNA","region":"Asia"},{"name":"Tampa, FL, United States","code":"TPA","region":"North America"},{"name":"Taipei, Taiwan","code":"TPE","region":"Asia"},{"name":"Tianjin, China","code":"TSN","region":"Asia"},{"name":"Berlin, Germany","code":"TXL","region":"Europe"},{"name":"Quito, Ecuador","code":"UIO","region":"Latin America & the Caribbean"},{"name":"Vienna, Austria","code":"VIE","region":"Europe"},{"name":"Warsaw, Poland","code":"WAW","region":"Europe"},{"name":"Wuhan, China","code":"WUH","region":"Asia"},{"name":"Wuxi, China","code":"WUX","region":"Asia"},{"name":"Xi'an, China","code":"XIY","region":"Asia"},{"name":"Montréal, QC, Canada","code":"YUL","region":"North America"},{"name":"Vancouver, BC, Canada","code":"YVR","region":"North America"},{"name":"Toronto, ON, Canada","code":"YYZ","region":"North America"},{"name":"Zagreb, Croatia","code":"ZAG","region":"Europe"},{"name":"Zürich, Switzerland","code":"ZRH","region":"Europe"}]`

var pops []pop
var popsByIDMap = make(map[string]pop)

func initPops() error {
	json.Unmarshal([]byte(popsJSON), &pops)
	for i, c := range pops {
		pops[i].Source = "built-in"
		c = pops[i]
		popsByIDMap[c.Code] = c
	}
	return nil
}

func getPop(popID string) *pop {
	if pop, ok := popsByIDMap[popID]; ok {
		return &pop
	}
	popID = strings.Split(popID, "-")[0]
	if pop, ok := popsByIDMap[popID]; ok {
		return &pop
	}
	if popID == "" {
		popID = "Unknown"
	}
	return &pop{
		Name:   "Unknown",
		Code:   popID,
		Region: "Unknown",
		Source: "fallback",
	}
}

func addPop(newP pop) {
	if _, ok := popsByIDMap[newP.Code]; ok {
		return
	}
	newP.Source = "external"
	pops = append(pops, newP)
	sort.Sort(byName(pops))
	popsByIDMap[newP.Code] = newP
}
