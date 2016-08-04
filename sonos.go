package sonos

import (
	"encoding/xml"
	"fmt"
	"github.com/franela/goreq"
	"strconv"
	"strings"
)

type TrackInfo struct {
	Title         string
	Artist        string
	Album         string
	AlbumArtUri   string
	Track         string
	TrackURI      string
	RelTime       string
	TrackDuration string
}

func NewZonePlayer(ipAddress string) ZonePlayer {
	return ZonePlayer{IpAddress: ipAddress}
}

type ZonePlayer struct {
	IpAddress string
}

func (zp *ZonePlayer) Play() {
	zp.sendCommand(TRANSPORT_ENDPOINT, PLAY_ACTION, PLAY_BODY)

}

func (zp *ZonePlayer) Pause() {
	zp.sendCommand(TRANSPORT_ENDPOINT, PAUSE_ACTION, PAUSE_BODY)
}

func (zp *ZonePlayer) Stop() {
	zp.sendCommand(TRANSPORT_ENDPOINT, STOP_ACTION, STOP_BODY)
}

func (zp *ZonePlayer) Next() {
	zp.sendCommand(TRANSPORT_ENDPOINT, NEXT_ACTION, NEXT_BODY)
}

func (zp *ZonePlayer) Previous() {
	zp.sendCommand(TRANSPORT_ENDPOINT, PAUSE_ACTION, PAUSE_BODY)
}

func (zp *ZonePlayer) GetVolume() int {
	_, res := zp.sendCommand(RENDERING_ENDPOINT, GET_VOLUME_ACTION, GET_VOLUME_BODY)

	level, err := strconv.Atoi(extractTagData("CurrentVolume", res))

	if err != nil {
		return -1
	}

	return level
}

func (zp *ZonePlayer) SetVolume(level int) (error, bool) {
	if level < 0 {
		level = 0
	}
	if level > 100 {
		level = 100
	}

	body := strings.Replace(SET_VOLUME_BODY_TEMPLATE, "{volume}", strconv.Itoa(level), 1)
	err, res := zp.sendCommand(RENDERING_ENDPOINT, SET_VOLUME_ACTION, body)

	return err, res == SET_VOLUME_RESPONSE
}

func (zp *ZonePlayer) CurrentTrackInfo() *TrackInfo {
	err, res := zp.sendCommand(TRANSPORT_ENDPOINT, GET_CUR_TRACK_ACTION, GET_CUR_TRACK_BODY)

	if err != nil {
		fmt.Printf("error: %v", err)
		return nil
	}

	type XmlGetPositionInfoResponse struct {
		XMLName       xml.Name `xml:"GetPositionInfoResponse"`
		Track         string
		RelTime       string
		TrackDuration string
		TrackMetaData string
	}

	type XmlBody struct {
		XMLName                 xml.Name                   `xml:"Body"`
		GetPositionInfoResponse XmlGetPositionInfoResponse `xml:"GetPositionInfoResponse"`
	}

	type XmlEnvelope struct {
		XMLName xml.Name `xml:"Envelope"`
		Body    XmlBody  `xml:"Body"`
	}

	type XmlItem struct {
		XMLName     xml.Name `xml:"item"`
		Title       string   `xml:"title"`
		Album       string   `xml:"album"`
		AlbumArtUri string   `xml:"albumArtURI"`
		Creator     string   `xml:"creator"`
	}

	type XmlTrackMetaData struct {
		XMLName xml.Name `xml:"DIDL-Lite"`
		Item    XmlItem  `xml:"item"`
	}

	fmt.Println(res)

	e := new(XmlEnvelope)

	err = xml.Unmarshal([]byte(res), &e)
	if err != nil {
		fmt.Printf("error: %v", err)
		return nil
	}

	//TODO: Clean this up properly
	tmdXml := strings.Replace(e.Body.GetPositionInfoResponse.TrackMetaData, "&quot;", "\"", -1)
	tmdXml = strings.Replace(tmdXml, "&gt;", ">", -1)
	tmdXml = strings.Replace(tmdXml, "&lt;", "<", -1)

	fmt.Println(tmdXml)
	tmd := new(XmlTrackMetaData)
	err = xml.Unmarshal([]byte(tmdXml), &tmd)
	if err != nil {
		fmt.Printf("err: %v", err)
		return nil
	}

	ti := &TrackInfo{Track: e.Body.GetPositionInfoResponse.Track,
		Title:       tmd.Item.Title,
		Album:       tmd.Item.Album,
		Artist:      tmd.Item.Creator,
		AlbumArtUri: "http://" + zp.IpAddress + ":1400" + tmd.Item.AlbumArtUri}

	return ti
}

func (zp *ZonePlayer) sendCommand(endPoint string, action string, body string) (error, string) {
	payload := strings.Replace(SOAP_TEMPLATE, "{body}", body, 1)

	req := goreq.Request{
		Method:      "POST",
		ContentType: "text/xml",
		Uri:         "http://" + zp.IpAddress + ":1400" + endPoint,
		Body:        payload,
	}
	req.AddHeader("SOAPACTION", action)
	res, err := req.Do()

	if err != nil {
		return err, ""
	}

	result, _ := res.Body.ToString()
	return err, result
}

//TODO: Find A Better Way
func extractTagData(tag string, xml string) string {
	openTag := "<" + tag + ">"
	closeTag := "</" + tag + ">"
	start := strings.Index(xml, openTag)
	end := strings.Index(xml, closeTag)

	if start == -1 || end == -1 {
		return ""
	}

	result := xml[start+len(openTag) : end]
	fmt.Println(result)

	return result
}

// definition section

const PLAYER_SEARCH = `M-SEARCH * HTTP/1.1
HOST: 239.255.255.250:reservedSSDPport
MAN: ssdp:discover
MX: 1
ST: urn:schemas-upnp-org:device:ZonePlayer:1`

const MCAST_GRP = "239.255.255.250"
const MCAST_PORT = 1900

const RADIO_STATIONS = 0
const RADIO_SHOWS = 1

const TRANSPORT_ENDPOINT = `/MediaRenderer/AVTransport/Control`
const RENDERING_ENDPOINT = `/MediaRenderer/RenderingControl/Control`
const DEVICE_ENDPOINT = `/DeviceProperties/Control`
const CONTENT_DIRECTORY_ENDPOINT = `/MediaServer/ContentDirectory/Control`

const ENQUEUE_BODY_TEMPLATE = `<u:SetAVTransportURI xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><CurrentURI>{uri}</CurrentURI><CurrentURIMetaData></CurrentURIMetaData></u:SetAVTransportURI>`
const ENQUEUE_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:SetAVTransportURIResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:SetAVTransportURIResponse></s:Body></s:Envelope>`

const PLAY_ACTION = `"urn:schemas-upnp-org:service:AVTransport:1#Play"`
const PLAY_BODY = `<u:Play xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><Speed>1</Speed></u:Play>`
const PLAY_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:PlayResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:PlayResponse></s:Body></s:Envelope>`

const PLAY_FROM_QUEUE_BODY_TEMPLATE = `
<u:SetAVTransportURI xmlns:u="urn:schemas-upnp-org:service:AVTransport:1">
    <InstanceID>0</InstanceID>
    <CurrentURI>{uri}</CurrentURI>
    <CurrentURIMetaData></CurrentURIMetaData>
</u:SetAVTransportURI>
`
const PLAY_FROM_QUEUE_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:SetAVTransportURIResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:SetAVTransportURIResponse></s:Body></s:Envelope>`

const PAUSE_ACTION = `"urn:schemas-upnp-org:service:AVTransport:1#Pause"`
const PAUSE_BODY = `<u:Pause xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><Speed>1</Speed></u:Pause>`
const PAUSE_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:PauseResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:PauseResponse></s:Body></s:Envelope>`

const STOP_ACTION = `"urn:schemas-upnp-org:service:AVTransport:1#Stop"`
const STOP_BODY = `<u:Stop xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><Speed>1</Speed></u:Stop>`
const STOP_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:StopResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:StopResponse></s:Body></s:Envelope>`

const NEXT_ACTION = `"urn:schemas-upnp-org:service:AVTransport:1#Next"`
const NEXT_BODY = `<u:Next xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><Speed>1</Speed></u:Next>`
const NEXT_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:NextResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:NextResponse></s:Body></s:Envelope>`

const PREV_ACTION = `"urn:schemas-upnp-org:service:AVTransport:1#Previous"`
const PREV_BODY = `<u:Previous xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><Speed>1</Speed></u:Previous>`
const PREV_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:PreviousResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:PreviousResponse></s:Body></s:Envelope>`

const MUTE_ACTION = `"urn:schemas-upnp-org:service:RenderingControl:1#SetMute"`
const MUTE_BODY_TEMPLATE = `<u:SetMute xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"><InstanceID>0</InstanceID><Channel>Master</Channel><DesiredMute>{mute}</DesiredMute></u:SetMute>`
const MUTE_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:SetMuteResponse xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"></u:SetMuteResponse></s:Body></s:Envelope>`

const GET_MUTE_ACTION = `"urn:schemas-upnp-org:service:RenderingControl:1#GetMute"`
const GET_MUTE_BODY = `<u:GetMute xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"><InstanceID>0</InstanceID><Channel>Master</Channel></u:GetMute>`

const SET_VOLUME_ACTION = `"urn:schemas-upnp-org:service:RenderingControl:1#SetVolume"`
const SET_VOLUME_BODY_TEMPLATE = `<u:SetVolume xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"><InstanceID>0</InstanceID><Channel>Master</Channel><DesiredVolume>{volume}</DesiredVolume></u:SetVolume>`
const SET_VOLUME_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:SetVolumeResponse xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"></u:SetVolumeResponse></s:Body></s:Envelope>`

const GET_VOLUME_ACTION = `"urn:schemas-upnp-org:service:RenderingControl:1#GetVolume"`
const GET_VOLUME_BODY = `<u:GetVolume xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"><InstanceID>0</InstanceID><Channel>Master</Channel></u:GetVolume>`

const SET_BASS_ACTION = `"urn:schemas-upnp-org:service:RenderingControl:1#SetBass"`
const SET_BASS_BODY_TEMPLATE = `<u:SetBass xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"><InstanceID>0</InstanceID><DesiredBass>{bass}</DesiredBass></u:SetBass>`
const SET_BASS_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:SetBassResponse xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"></u:SetBassResponse></s:Body></s:Envelope>`

const GET_BASS_ACTION = `"urn:schemas-upnp-org:service:RenderingControl:1#GetBass"`
const GET_BASS_BODY = `<u:GetBass xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"><InstanceID>0</InstanceID><Channel>Master</Channel></u:GetBass>`

const SET_TREBLE_ACTION = `"urn:schemas-upnp-org:service:RenderingControl:1#SetTreble"`
const SET_TREBLE_BODY_TEMPLATE = `<u:SetTreble xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"><InstanceID>0</InstanceID><DesiredTreble>{treble}</DesiredTreble></u:SetTreble>`
const SET_TREBLE_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:SetTrebleResponse xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"></u:SetTrebleResponse></s:Body></s:Envelope>`

const GET_TREBLE_ACTION = `"urn:schemas-upnp-org:service:RenderingControl:1#GetTreble"`
const GET_TREBLE_BODY = `<u:GetTreble xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"><InstanceID>0</InstanceID><Channel>Master</Channel></u:GetTreble>`

const SET_LOUDNESS_ACTION = `"urn:schemas-upnp-org:service:RenderingControl:1#SetLoudness"`
const SET_LOUDNESS_BODY_TEMPLATE = `<u:SetLoudness xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"><InstanceID>0</InstanceID><Channel>Master</Channel><DesiredLoudness>{loudness}</DesiredLoudness></u:SetLoudness>`
const SET_LOUDNESS_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:SetLoudnessResponse xmlns:u="urn:schemas-upnp-org:service:RenderingControl:1"></u:SetLoudnessResponse></s:Body></s:Envelope>`

const SET_TRANSPORT_ACTION = `"urn:schemas-upnp-org:service:AVTransport:1#SetAVTransportURI"`

const JOIN_BODY_TEMPLATE = `<u:SetAVTransportURI xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><CurrentURI>x-rincon:{master_uid}</CurrentURI><CurrentURIMetaData></CurrentURIMetaData></u:SetAVTransportURI>`
const JOIN_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:SetAVTransportURIResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:SetAVTransportURIResponse></s:Body></s:Envelope>`

const UNJOIN_ACTION = `"urn:schemas-upnp-org:service:AVTransport:1#BecomeCoordinatorOfStandaloneGroup"`
const UNJOIN_BODY = `<u:BecomeCoordinatorOfStandaloneGroup xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><Speed>1</Speed></u:BecomeCoordinatorOfStandaloneGroup>`
const UNJOIN_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:BecomeCoordinatorOfStandaloneGroupResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:BecomeCoordinatorOfStandaloneGroupResponse></s:Body></s:Envelope>`

const SET_LINEIN_BODY_TEMPLATE = `<u:SetAVTransportURI xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><CurrentURI>x-rincon-stream:{speaker_uid}</CurrentURI><CurrentURIMetaData></CurrentURIMetaData></u:SetAVTransportURI>`
const SET_LINEIN_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:SetAVTransportURIResponse xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"></u:SetAVTransportURIResponse></s:Body></s:Envelope>`

const SET_LEDSTATE_ACTION = `"urn:schemas-upnp-org:service:DeviceProperties:1#SetLEDState"`
const SET_LEDSTATE_BODY_TEMPLATE = `<u:SetLEDState xmlns:u="urn:schemas-upnp-org:service:DeviceProperties:1"><DesiredLEDState>{state}</DesiredLEDState>`
const SET_LEDSTATE_RESPONSE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:SetLEDStateResponse xmlns:u="urn:schemas-upnp-org:service:DeviceProperties:1"></u:SetLEDStateResponse></s:Body></s:Envelope>`

const GET_CUR_TRACK_ACTION = `"urn:schemas-upnp-org:service:AVTransport:1#GetPositionInfo"`
const GET_CUR_TRACK_BODY = `<u:GetPositionInfo xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><Channel>Master</Channel></u:GetPositionInfo>`

const GET_CUR_TRANSPORT_ACTION = `"urn:schema-upnp-org:service:AVTransport:1#GetTransportInfo"`
const GET_CUR_TRANSPORT_BODY = `<u:GetTransportInfo xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID></u:GetTransportInfo></s:Body></s:Envelope>`

const SOAP_TEMPLATE = `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body>{body}</s:Body></s:Envelope>`

const SEEK_ACTION = `"urn:schemas-upnp-org:service:AVTransport:1#Seek"`
const SEEK_TRACK_BODY_TEMPLATE = `
<u:Seek xmlns:u="urn:schemas-upnp-org:service:AVTransport:1">
<InstanceID>0</InstanceID>
<Unit>TRACK_NR</Unit>
<Target>{track}</Target>
</u:Seek>
`

const SEEK_TIMESTAMP_BODY_TEMPLATE = `<u:Seek xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><Unit>REL_TIME</Unit><Target>{timestamp}</Target></u:Seek>`

const PLAY_URI_BODY_TEMPLATE = `<u:SetAVTransportURI xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><CurrentURI>{uri}</CurrentURI><CurrentURIMetaData>{meta}</CurrentURIMetaData></u:SetAVTransportURI>`

const GET_QUEUE_BODY_TEMPLATE = `<u:Browse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"><ObjectID>Q:0</ObjectID><BrowseFlag>BrowseDirectChildren</BrowseFlag><Filter>dc:title,res,dc:creator,upnp:artist,upnp:album,upnp:albumArtURI</Filter><StartingIndex>{0}</StartingIndex><RequestedCount>{1}</RequestedCount><SortCriteria></SortCriteria></u:Browse>`

const ADD_TO_QUEUE_ACTION = `urn:schemas-upnp-org:service:AVTransport:1#AddURIToQueue`
const ADD_TO_QUEUE_BODY_TEMPLATE = `<u:AddURIToQueue xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID><EnqueuedURI>{uri}</EnqueuedURI><EnqueuedURIMetaData></EnqueuedURIMetaData><DesiredFirstTrackNumberEnqueued>0</DesiredFirstTrackNumberEnqueued><EnqueueAsNext>1</EnqueueAsNext></u:AddURIToQueue>`

const REMOVE_FROM_QUEUE_ACTION = `urn:schemas-upnp-org:service:AVTransport:1#RemoveTrackFromQueue`
const REMOVE_FROM_QUEUE_BODY_TEMPLATE = `<u:RemoveTrackFromQueue xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>{instance}</InstanceID><ObjectID>{objid}</ObjectID><UpdateID>{updateid}</UpdateID></u:RemoveTrackFromQueue>`

const CLEAR_QUEUE_ACTION = `"urn:schemas-upnp-org:service:AVTransport:1#RemoveAllTracksFromQueue"`
const CLEAR_QUEUE_BODY = `<u:RemoveAllTracksFromQueue xmlns:u="urn:schemas-upnp-org:service:AVTransport:1"><InstanceID>0</InstanceID></u:RemoveAllTracksFromQueue>`

const BROWSE_ACTION = `"urn:schemas-upnp-org:service:ContentDirectory:1#Browse"`
const GET_RADIO_FAVORITES_BODY_TEMPLATE = `<u:Browse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"><ObjectID>R:0/{0}</ObjectID><BrowseFlag>BrowseDirectChildren</BrowseFlag><Filter>dc:title,res,dc:creator,upnp:artist,upnp:album,upnp:albumArtURI</Filter><StartingIndex>{1}</StartingIndex><RequestedCount>{2}</RequestedCount><SortCriteria/></u:Browse>`

const SET_PLAYER_NAME_ACTION = `"urn:schemas-upnp-org:service:DeviceProperties:1#SetZoneAttributes"`
const SET_PLAYER_NAME_BODY_TEMPLATE = `"<u:SetZoneAttributes xmlns:u="urn:schemas-upnp-org:service:DeviceProperties:1"><DesiredZoneName>{playername}</DesiredZoneName><DesiredIcon /><DesiredConfiguration /></u:SetZoneAttributes>"`
const SET_PLAYER_NAME_RESPONSE = `"<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:SetZoneAttributesResponse xmlns:u="urn:schemas-upnp-org:service:DeviceProperties:1"></u:SetZoneAttributesResponse></s:Body></s:Envelope>"`
