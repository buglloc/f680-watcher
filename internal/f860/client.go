package f860

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

type RouterConfig struct {
	Upstream string
	Username string
	Password string
}

type Client struct {
	httpc        *resty.Client
	routerConfig RouterConfig
}

var sessionTokenRe = regexp.MustCompile(`_sessionTmpToken = "([\\x0-9A-Fa-f]+)"`)

func NewClient(routerConfig RouterConfig, opts ...Option) (*Client, error) {
	encKey, err := ParseEncryptionKey(defaultEncryptionKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse encryption key: %w", err)
	}

	out := &Client{
		httpc: resty.New().
			SetBaseURL(routerConfig.Upstream).
			SetPreRequestHook(newRestySignHook(encKey)),
		routerConfig: routerConfig,
	}

	for _, opt := range opts {
		opt(out)
	}

	return out, out.Reset()
}

func (c *Client) Reset() error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("unable to create new cookie jar: %w", err)
	}

	c.httpc.SetCookieJar(jar)
	return nil
}

func (c *Client) Login(ctx context.Context) (bool, error) {
	sessionToken, err := c.loginSessionToken(ctx)
	if err != nil {
		return false, fmt.Errorf("unable to get new session: %w", err)
	}

	loginToken, err := c.loginToken(ctx)
	if err != nil {
		return false, fmt.Errorf("unable to get login token: %w", err)
	}

	ok, err := c.authorize(
		ctx,
		sessionToken, loginToken,
		c.routerConfig.Username, c.routerConfig.Password,
	)
	if !ok {
		return false, err
	}

	return true, nil
}

func (c *Client) LanDevDHCPSources(ctx context.Context) ([]DevDHCPSource, error) {
	if _, err := c.prepareLanMgr(ctx); err != nil {
		return nil, fmt.Errorf("unable to prepare lan mngr: %w", err)
	}

	var ajaxRsp struct {
		BaseAjaxRsp
		LanDevDHCPSourceID struct {
			Instance []struct {
				ParaName  []string `xml:"ParaName"`
				ParaValue []string `xml:"ParaValue"`
			} `xml:"Instance"`
		} `xml:"OBJ_LANDEVDHCPSOURCE_ID"`
	}
	rsp, err := c.httpc.R().
		SetResult(&ajaxRsp).
		SetContext(ctx).
		Get("/?_type=menuData&_tag=Localnet_LanDevDHCPSource_lua.lua")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if !rsp.IsSuccess() {
		return nil, fmt.Errorf("non-200 status code: %s", string(rsp.Body()))
	}

	if err := ajaxRsp.RemoteError(); err != nil {
		return nil, err
	}

	out := make([]DevDHCPSource, len(ajaxRsp.LanDevDHCPSourceID.Instance))
	for i, instance := range ajaxRsp.LanDevDHCPSourceID.Instance {
		var source DevDHCPSource
		for k, para := range instance.ParaName {
			switch para {
			case "_InstID":
				source.ID = instance.ParaValue[k]
			case "ProcFlag":
				_ = source.ProcFlag.FromRouter(instance.ParaValue[k])
			case "VendorClassID":
				source.VendorClassID = instance.ParaValue[k]
			default:
				log.Warn().Str("para_name", para).Msg("unsupported LanDevDHCPSource instance param")
			}
		}

		out[i] = source
	}

	return out, nil
}

func (c *Client) UpdateLanDevDHCPSource(ctx context.Context, sources ...DevDHCPSource) error {
	if len(sources) == 0 {
		return nil
	}

	sessionToken, err := c.prepareLanMgr(ctx)
	if err != nil {
		return fmt.Errorf("unable to prepare lan mngr: %w", err)
	}

	formValues := url.Values{
		"IF_ACTION":     {"Apply"},
		"_InstNum":      {strconv.Itoa(len(sources))},
		"_sessionTOKEN": {sessionToken},
	}

	for i, source := range sources {
		formValues.Set(
			fmt.Sprintf("_InstID_%d", i),
			source.ID,
		)

		formValues.Set(
			fmt.Sprintf("ProcFlag_%d", i),
			source.ProcFlag.Router(),
		)
	}

	var ajaxRsp BaseAjaxRsp
	rsp, err := c.httpc.R().
		SetFormDataFromValues(formValues).
		SetResult(&ajaxRsp).
		SetHeaders(map[string]string{
			"X-Requested-With": "XMLHttpRequest",
			"Origin":           "http://172.16.1.1",
			"Referer":          "http://172.16.1.1/",
		}).
		SetContext(ctx).
		Post("/?_type=menuData&_tag=Localnet_LanDevDHCPSource_lua.lua")
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	if !rsp.IsSuccess() {
		return fmt.Errorf("non-200 status code: %s", string(rsp.Body()))
	}

	if err := ajaxRsp.RemoteError(); err != nil {
		return err
	}

	return nil
}

func (c *Client) prepareLanMgr(ctx context.Context) (string, error) {
	rsp, err := c.httpc.R().
		SetContext(ctx).
		Get("/?_type=menuView&_tag=lanMgrIpv4&Menu3Location=0")
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}

	if !rsp.IsSuccess() {
		return "", fmt.Errorf("non-200 status code: %s", string(rsp.Body()))
	}

	matches := sessionTokenRe.FindSubmatch(rsp.Body())
	if len(matches) != 2 {
		return "", nil
	}

	hexToken := string(matches[1])
	rawToken, err := hex.DecodeString(strings.ReplaceAll(hexToken, `\x`, ""))
	if err != nil {
		return "", fmt.Errorf("invalid session token %q: %w", hexToken, err)
	}
	return string(rawToken), nil
}

func (c *Client) loginSessionToken(ctx context.Context) (string, error) {
	var loginRsp struct {
		SessToken string `json:"sess_token"`
	}
	rsp, err := c.httpc.R().
		SetResult(&loginRsp).
		SetContext(ctx).
		Get("/?_type=loginData&_tag=login_entry")
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}

	if !rsp.IsSuccess() {
		return "", fmt.Errorf("non-200 status code: %s", string(rsp.Body()))
	}

	return loginRsp.SessToken, nil
}

func (c *Client) loginToken(ctx context.Context) (string, error) {
	var ajaxRsp struct {
		XMLName    xml.Name `xml:"ajax_response_xml_root"`
		LoginToken string   `xml:",chardata"`
	}
	rsp, err := c.httpc.R().
		SetResult(&ajaxRsp).
		SetContext(ctx).
		Get("/?_type=loginData&_tag=login_token")
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}

	if !rsp.IsSuccess() {
		return "", fmt.Errorf("non-200 status code: %s", string(rsp.Body()))
	}

	return strings.TrimSpace(ajaxRsp.LoginToken), nil
}

func (c *Client) authorize(ctx context.Context, sessionToken, loginToken, username, password string) (bool, error) {
	passwordHash := sha256.New()
	passwordHash.Write([]byte(password + loginToken))

	var loginRsp struct {
		LockingTime *int   `json:"lockingTime"`
		SessToken   string `json:"sess_token"`
	}
	rsp, err := c.httpc.R().
		SetFormData(map[string]string{
			"action":        "login",
			"Username":      username,
			"Password":      hex.EncodeToString(passwordHash.Sum(nil)),
			"_sessionTOKEN": sessionToken,
		}).
		SetResult(&loginRsp).
		SetContext(ctx).
		Post("/?_type=loginData&_tag=login_entry")
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}

	if !rsp.IsSuccess() {
		return false, fmt.Errorf("non-200 status code: %s", string(rsp.Body()))
	}

	return loginRsp.LockingTime == nil, nil
}

func newRestySignHook(key *EncryptionKey) resty.PreRequestHook {
	return func(_ *resty.Client, req *http.Request) error {
		if req.Method != http.MethodPost {
			return nil
		}

		if req.GetBody == nil {
			return errors.New("request func GetBody is not set")
		}

		body, err := req.GetBody()
		if err != nil {
			return fmt.Errorf("request GetBody error: %w", err)
		}

		if body == nil {
			return nil
		}

		payload, err := io.ReadAll(body)
		if err != nil {
			return fmt.Errorf("unable to read body for signing: %w", err)
		}

		payloadHasher := sha256.New()
		payloadHasher.Write(payload)
		payloadHash := payloadHasher.Sum(nil)
		digest := make([]byte, hex.EncodedLen(len(payloadHash)))
		hex.Encode(digest, payloadHash)
		encryptedPayload, err := key.Encrypt(digest)
		if err != nil {
			return fmt.Errorf("encryption failed: %w", err)
		}

		req.Header.Set("Check", base64.StdEncoding.EncodeToString(encryptedPayload))
		return nil
	}
}
