package WebMap

import (
	"errors"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"encoding/json"
)

/* http://lbsyun.baidu.com/apiconsole/key */
type MapBD struct {
	Token	string
}

type convertLL struct {
	Status	int			`json:"status"`
	X		float32 	`json:"x"`
	Y		float32 	`json:"y"`
}

func (this *MapBD) ConvertLL(Lon, Lat float32) (float32, float32, error) {
	URL := "http://api.map.baidu.com/geoconv/v1/?ak=%s&coords=%f,%f&output=json"
	URL = fmt.Sprintf(URL, this.Token, Lon, Lat)
	
	Resp, err := http.Get(URL)
	defer Resp.Body.Close()
	if err != nil { return 0.0, 0.0, err }

	Body, err := ioutil.ReadAll(Resp.Body)
	if err != nil { return 0.0, 0.0, err }

	Result := &convertLL{}
	err = json.Unmarshal(Body, &Result)
	if err != nil { return 0.0, 0.0, err }
	if Result.Status != 0 { return 0.0, 0.0, errors.New("Result Error")}

	S := Body[bytes.IndexByte(Body, '[') + 1 : bytes.IndexByte(Body, ']')]
	err = json.Unmarshal(S, &Result)
	if err != nil { return 0.0, 0.0, err }
	
	return Result.X, Result.Y, nil
}

func (this *MapBD) LocationLL(Lon, Lat float32) (string, error) {
	URL := "http://api.map.baidu.com/geocoder/v2/?ak=%s&callback=renderReverse&location=%f,%f&output=json&pois=1"
	URL = fmt.Sprintf(URL, this.Token, Lat, Lon)
	fmt.Println(URL)

	Resp, err := http.Get(URL)
	defer Resp.Body.Close()
	if err != nil { return "", err }

	Body, err := ioutil.ReadAll(Resp.Body)
	if err != nil { return "", err }

	S := Body[bytes.IndexByte(Body, '(') + 1 : bytes.IndexByte(Body, ')')]
	if bytes.Index(S, []byte("\"status\":0,")) < 0 { return "", errors.New("Result Error") }

	S = S[bytes.Index(S, []byte("formatted_address")) + 20 : len(S) - 1]
	S = S[ : bytes.IndexByte(S, '"')]

	return string(S), nil
}
