package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"

	"code.google.com/p/go.net/publicsuffix"
	"golang.org/x/crypto/nacl/secretbox"
)

var pad = []byte(" super jumpy something jumps all over ")
var storage = ".redcapfs_tokens"

const keySize = 32

// TokenStoreRemove clears all stored tokens
func TokenStoreRemove(passPhase string) {
	// only allow to delete the tokens file if you have the correct password to read the file
	_ = TokenStoreGet(passPhase)

	err := os.Remove(storage)
	if err != nil {
		fmt.Println("Error: could not remove the stored tokens file (.redcapfs_tokens)")
	}
}

// TokenStorePut saves the current tokens, it requires a pass-phrase
func TokenStorePut(passPhrase string, data map[string][]string) {

	key := []byte(passPhrase)
	key = append(key, pad...)
	secretKey := new([keySize]byte)
	copy(secretKey[:], key[:keySize])

	// You must use a different nonce for each message you encrypt with the
	// same key. Since the nonce here is 192 bits long, a random value
	// provides a sufficiently small probability of repeats.
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		panic(err)
	}
	rep, err := json.Marshal(data)

	// This encrypts data and appends the result to the nonce.
	encrypted := secretbox.Seal(nonce[:], rep, &nonce, secretKey)

	err = ioutil.WriteFile(storage, encrypted, 0644)
	if err != nil {
		panic(err)
	}
}

// TokenStoreGet returns the current tokens, it requires a pass-phrase
func TokenStoreGet(passPhrase string) map[string][]string {

	key := []byte(passPhrase)
	key = append(key, pad...)
	secretKey := new([keySize]byte)
	copy(secretKey[:], key[:keySize])

	encrypted, err := ioutil.ReadFile(storage)
	if err != nil {
		fmt.Println("Error: could not read tokens from file, assume the file does not exist yet, create empty entries")
		t := make(map[string][]string, 0)
		t["accessTokens"] = make([]string, 0)
		t["REDCapURL"] = make([]string, 0)
		t["REDCapURL"] = append(t["REDCapURL"], "https://abcd-rc.ucsd.edu/redcap/api/")
		return t
	}

	// When you decrypt, you must use the same nonce and key you used to
	// encrypt the message. One way to achieve this is to store the nonce
	// alongside the encrypted message. Above, we stored the nonce in the first
	// 24 bytes of the encrypted text.
	var decryptNonce [24]byte
	copy(decryptNonce[:], encrypted[:24])
	decrypted, ok := secretbox.Open([]byte{}, encrypted[24:], &decryptNonce, secretKey)
	if !ok {
		panic("decryption error")
	}

	var msg map[string][]string
	err = json.Unmarshal(decrypted, &msg)
	if err != nil {
		panic(err)
	}
	// check if we have a REDCapURL value, if it does not exist add one
	if _, ok := msg["REDCapURL"]; !ok {
		msg["REDCapURL"] = make([]string, 0)
		msg["REDCapURL"] = append(msg["REDCapURL"], "https://abcd-rc.ucsd.edu/redcap/api/")
	}

	return msg
}

// GetParticipantsBySite will ask REDCap about the list of participants
func GetParticipantsBySite(tokens map[string][]string) []map[string]string {
	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Fatal(err)
	}
	client := http.Client{Jar: jar}
	REDCapURL := tokens["REDCapURL"][0]

	var ret []map[string]string
	for _, token := range tokens["accessTokens"] {
		values := url.Values{}
		values.Set("token", token)
		values.Add("content", "record")
		values.Add("format", "json")
		values.Add("type", "flat")
		values.Add("fields[0]", "enroll_total")
		values.Add("fields[1]", "id_redcap")
		values.Add("fields[2]", "cp_timestamp_v2")
		values.Add("events[0]", "baseline_year_1_arm_1")
		values.Add("rawOrLabel", "raw")
		values.Add("rawOrLabelHeaders", "raw")
		values.Add("exportCheckboxLabel", "false")
		values.Add("exportSurveyFields", "false")
		values.Add("exportDataAccessGroups", "true")
		values.Add("returnFormat", "json")

		req, err := http.NewRequest("POST", REDCapURL, bytes.NewBufferString(values.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(values.Encode())))

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		data, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		var dat []map[string]string
		if err = json.Unmarshal(data, &dat); err != nil {
			panic(err)
		}

		// create array of strings from list
		for _, elem := range dat {
			if elem["enroll_total___1"] == "1" {
				ret = append(ret, elem)
			}
		}
	}
	return ret
}

// GetInstruments returns the list of all instruments from REDCap
func GetInstruments(tokens map[string][]string) []map[string]string {
	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Fatal(err)
	}
	client := http.Client{Jar: jar}
	REDCapURL := tokens["REDCapURL"][0]

	var dat []map[string]string
	for _, token := range tokens["accessTokens"] {
		values := url.Values{}
		values.Set("token", token)
		values.Add("content", "metadata")
		values.Add("format", "json")
		values.Add("returnFormat", "json")

		req, err := http.NewRequest("POST", REDCapURL, bytes.NewBufferString(values.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(values.Encode())))

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		data, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		if err = json.Unmarshal(data, &dat); err != nil {
			panic(err)
		}
		// we only need to call this once, return the value obtained from the first token
		return dat
	}
	return dat
}

// GetFormEventMapping returns the events and forms in an array
func GetFormEventMapping(tokens map[string][]string) []map[string]string {

	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Fatal(err)
	}
	client := http.Client{Jar: jar}

	REDCapURL := tokens["REDCapURL"][0]

	var dat2 []map[string]string
	for _, token := range tokens["accessTokens"] {
		values := url.Values{}
		values.Set("token", token)
		values.Add("content", "formEventMapping")
		values.Add("format", "json")
		values.Add("returnFormat", "json")

		req, err := http.NewRequest("POST", REDCapURL, bytes.NewBufferString(values.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(values.Encode())))

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		data, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		var dat []map[string]interface{}
		if err = json.Unmarshal(data, &dat); err != nil {
			panic(err)
		}
		for _, v := range dat {
			d := make(map[string]string, 2)
			for k, v2 := range v {
				if k == "unique_event_name" {
					d[k] = v2.(string)
				}
				if k == "form" {
					d["form"] = v2.(string)
				}
			}
			dat2 = append(dat2, d)
		}
		return dat2 // return the value obtained from the first token
	}
	return dat2
}

// GetDataDictionary returns the data dictioanry for the given list of instruments
func GetDataDictionary(instruments []string, tokens map[string][]string) []map[string]string {
	// we will only use the first token for this - not all the tokens we could - data dictionaries are the same regardless of account

	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Fatal(err)
	}
	client := http.Client{Jar: jar}

	/*vals, err := json.Marshal(tokens)
	if err != nil {
		fmt.Println("Error in converting token list to string")
	}
	fmt.Println("In getInstrument, looking at these tokens", string(vals)) */
	var dat []map[string]string
	if len(tokens["REDCapURL"]) < 1 {
		return dat
	}
	REDCapURL := tokens["REDCapURL"][0]

	token := tokens["accessTokens"][0]
	values := url.Values{}
	values.Set("token", token)
	values.Add("content", "metadata")
	values.Add("format", "json")
	values.Add("returnFormat", "json")
	for i, instrument := range instruments {
		values.Add("forms["+strconv.Itoa(i)+"]", instrument)
	}

	req, err := http.NewRequest("POST", REDCapURL, bytes.NewBufferString(values.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(values.Encode())))

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	if err = json.Unmarshal(data, &dat); err != nil {
		panic(err)
	}
	return dat
}

// GetInstrument returns the values for a single instrument
func GetInstrument(instrument string, tokens map[string][]string) []map[string]string {

	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Fatal(err)
	}
	client := http.Client{Jar: jar}

	/*vals, err := json.Marshal(tokens)
	if err != nil {
		fmt.Println("Error in converting token list to string")
	}
	fmt.Println("In getInstrument, looking at these tokens", string(vals)) */

	REDCapURL := tokens["REDCapURL"][0]

	var ret []map[string]string
	for _, token := range tokens["accessTokens"] {
		values := url.Values{}
		values.Set("token", token)
		values.Add("content", "record")
		values.Add("format", "json")
		values.Add("type", "flat")
		values.Add("forms[0]", instrument)
		values.Add("fields[0]", "id_redcap")
		values.Add("fields[1]", "enroll_total")
		values.Add("rawOrLabel", "raw")
		values.Add("rawOrLabelHeaders", "raw")
		values.Add("exportCheckboxLabel", "false")
		values.Add("exportSurveyFields", "false")
		values.Add("exportDataAccessGroups", "true") // this will only return something if the user has access to more than one data access group
		values.Add("returnFormat", "json")

		req, err := http.NewRequest("POST", REDCapURL, bytes.NewBufferString(values.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(values.Encode())))

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		data, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		var dat []map[string]string
		if err = json.Unmarshal(data, &dat); err != nil {
			panic(err)
		}
		// create array of strings from list
		for _, elem := range dat {
			if elem["enroll_total___1"] == "1" {
				ret = append(ret, elem)
			}
		}
	}
	return ret
}

// GetMeasure returns a single measure
func GetMeasure(measure string, tokens map[string][]string) []map[string]string {

	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Fatal(err)
	}
	client := http.Client{Jar: jar}
	REDCapURL := tokens["REDCapURL"][0]

	var ret []map[string]string
	for _, token := range tokens["accessTokens"] {
		fmt.Println("Start with token: ", token)
		values := url.Values{}
		values.Set("token", token)
		values.Add("content", "record")
		values.Add("format", "json")
		values.Add("type", "flat")
		values.Add("fields[0]", measure)
		values.Add("fields[1]", "id_redcap")
		values.Add("fields[2]", "enroll_total")
		values.Add("rawOrLabel", "raw")
		values.Add("rawOrLabelHeaders", "raw")
		values.Add("exportCheckboxLabel", "false")
		values.Add("exportSurveyFields", "false")
		values.Add("exportDataAccessGroups", "true")
		values.Add("returnFormat", "json")

		req, err := http.NewRequest("POST", REDCapURL, bytes.NewBufferString(values.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(values.Encode())))

		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		data, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		var dat []map[string]string
		if err = json.Unmarshal(data, &dat); err != nil {
			panic(err)
		}
		// create array of strings from list
		for _, elem := range dat {
			if elem["enroll_total___1"] == "1" {
				ret = append(ret, elem)
			}
		}
	}
	return ret
}
