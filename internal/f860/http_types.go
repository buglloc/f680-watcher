package f860

import (
	"encoding/xml"
	"fmt"
)

type DevDHCPSource struct {
	ID            string
	ProcFlag      DHCPSourceKind
	VendorClassID string
}

type BaseAjaxRsp struct {
	XMLName xml.Name `xml:"ajax_response_xml_root"`
	ErrorID struct {
		XMLName xml.Name `xml:"IF_ERRORID"`
		Value   string   `xml:",chardata"`
	}
	ErrorStr struct {
		XMLName xml.Name `xml:"IF_ERRORSTR"`
		Value   string   `xml:",chardata"`
	}
}

func (r *BaseAjaxRsp) RemoteError() error {
	if r.ErrorID.Value == "0" {
		return nil
	}

	if r.ErrorStr.Value == "SessionTimeout" {
		return &ErrUnauthorized{}
	}

	return fmt.Errorf("remote error: %s", r.ErrorStr.Value)
}
