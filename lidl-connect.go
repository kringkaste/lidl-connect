package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"time"

	"github.com/m7shapan/njson"
	"github.com/spf13/viper"
)

type Configuration struct {
	Grant_type    string `json:"grant_type"`
	Client_id     string `json:"client_id"`
	Client_secret string `json:"client_secret"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

type Token struct {
	Grant_type    string `json:"grant_type"`
	Expires_in    int    `json:"expires_in"`
	Access_token  string `json:"access_token"`
	Refresh_token string `json:"refresh_token"`
}

type Consumption struct {
	Consumed        uint      `json:"consumed"`
	Left            uint      `json:"left"`
	Max             uint      `json:"max"`
	Unit            string    `json:"unit"`
	Type            string    `json:"type"`
	ExpirationDate  time.Time `json:"expirationDate"`
	ConsumedPercent float64
	LeftPercent     float64
	DaysLeft        float64
}

type Consumptions struct {
	Consumptions []Consumption `njson:"data.consumptions.consumptionsForUnit"`
}

type Balance struct {
	Balance uint `njson:"data.currentCustomer.balance"`
}

type Tariff struct {
	Name      string    `njson:"data.tariffs.bookedTariff.name"`
	Fee       uint      `njson:"data.tariffs.bookedTariff.basicFee"`
	RenewDate time.Time `njson:"data.tariffs.bookedTariff.renewContractDate"`
}

type Output struct {
	Consumptions []Consumption
	Balance      uint
	Tariff       Tariff
}

func round(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func getConfiguration() Configuration {
	viper.SetDefault("client_id", "lidl")
	viper.SetDefault("client_secret", "lidl")
	viper.SetDefault("grant_type", "password")
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath("$HOME/.config/lidl-connect")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	config := Configuration{}
	config.Grant_type = viper.GetString("grant_type")
	config.Client_id = viper.GetString("client_id")
	config.Client_secret = viper.GetString("client_secret")
	config.Username = viper.GetString("username")
	config.Password = viper.GetString("password")

	return config
}

func getToken(config Configuration) Token {
	apiUrl := "https://api.lidl-connect.de/api/token"

	jsonData, _ := json.Marshal(config)

	request, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	token := Token{}
	json.Unmarshal([]byte(body), &token)

	return token
}

func getConsumption(token Token) Consumptions {
	apiUrl := "https://api.lidl-connect.de/api/graphql"

	var jsonData = []byte(`{
		"operation": "consumptions",
		"variables": {},
		"query": "query consumptions {\n  consumptions {\n    consumptionsForUnit {\n      consumed\n      unit\n      formattedUnit\n      type\n      description\n      expirationDate\n      left\n      max\n      __typename\n    }\n    __typename\n  }\n}\n"
	}`)

	request, error := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("Authorization", "Bearer "+token.Access_token)

	client := &http.Client{}
	response, error := client.Do(request)
	if error != nil {
		panic(error)
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	data := Consumptions{}
	njson.Unmarshal([]byte(body), &data)

	date := time.Now()

	for i, consumption := range data.Consumptions {
		data.Consumptions[i].LeftPercent = round(float64(consumption.Left)/float64(consumption.Max)*100, 0)
		data.Consumptions[i].ConsumedPercent = round(float64(consumption.Consumed)/float64(consumption.Max)*100, 0)
		diff := consumption.ExpirationDate.Sub(date)
		data.Consumptions[i].DaysLeft = round(diff.Hours()/24, 0)
	}

	return data
}

func getBalance(token Token) Balance {
	apiUrl := "https://api.lidl-connect.de/api/graphql"

	var jsonData = []byte(`{
		"operation": "balanceInfo",
		"variables": {},
		"query": "query balanceInfo {\n  currentCustomer {\n    balance\n    __typename\n  }\n}\n"
	}`)

	request, error := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("Authorization", "Bearer "+token.Access_token)

	client := &http.Client{}
	response, error := client.Do(request)
	if error != nil {
		panic(error)
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	data := Balance{}
	njson.Unmarshal([]byte(body), &data)

	return data
}

func getTariff(token Token) Tariff {
	apiUrl := "https://api.lidl-connect.de/api/graphql"

	var jsonData = []byte(`{
		"operation": "bookedTariff",
		"variables": {},
		"query": "query bookedTariff {\n  tariffs {\n    bookableTariffs {\n      bookableTariffs {\n        tariffId\n        __typename\n      }\n      __typename\n    }\n    bookedTariff {\n      tariffId\n      name\n      basicFee\n      statusKey\n      smsFlat\n      cancelable\n      tariffChangePossible\n      isPendingTariff\n      isSuspendedActive\n      phoneFlat\n      tariffState\n      renewContractDate\n      possibleChangingDate\n      terminationDate\n      changeTariffDate\n      runtime {\n        amount\n        unit\n        __typename\n      }\n      __typename\n    }\n    pendingTariff {\n      tariffId\n      name\n      tariffState\n      basicFee\n      tariffChangePossible\n      isPendingTariff\n      isSuspendedActive\n      cancelable\n      statusKey\n      smsFlat\n      phoneFlat\n      phoneFlat\n      tariffState\n      renewContractDate\n      possibleChangingDate\n      terminationDate\n      changeTariffDate\n      runtime {\n        amount\n        unit\n        __typename\n      }\n      __typename\n    }\n    __typename\n  }\n}\n"
	}`)

	request, error := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Set("Authorization", "Bearer "+token.Access_token)

	client := &http.Client{}
	response, error := client.Do(request)
	if error != nil {
		panic(error)
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)
	data := Tariff{}
	njson.Unmarshal([]byte(body), &data)

	return data
}

func main() {
	config := getConfiguration()
	token := getToken(config)

	output := Output{}
	output.Consumptions = getConsumption(token).Consumptions
	output.Balance = getBalance(token).Balance
	output.Tariff = getTariff(token)

	json, _ := json.Marshal(output)
	fmt.Printf("%s\n", json)
}
