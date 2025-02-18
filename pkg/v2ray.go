package thirdparty

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/v2fly/v2ray-core/v4/app/proxyman/command"
	statsservice "github.com/v2fly/v2ray-core/v4/app/stats/command"
	"github.com/v2fly/v2ray-core/v4/common/protocol"
	"github.com/v2fly/v2ray-core/v4/common/serial"
	"github.com/v2fly/v2ray-core/v4/proxy/vmess"
	helper "github.com/xvv6u577/logv2fs/helpers"
	"github.com/xvv6u577/logv2fs/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const (
	ALTERID = 64
	LEVEL   = 0
	PLAIN   = "plain"
	DELETE  = "delete"
	OVERDUE = "overdue"
)

type (
	User = model.UserTrafficLogs
)

type StatsServiceClient struct {
	statsservice.StatsServiceClient
}

type HandlerServiceClient struct {
	command.HandlerServiceClient
	inboundTag string
}

func NewStatsServiceClient(client *grpc.ClientConn) *StatsServiceClient {
	return &StatsServiceClient{
		StatsServiceClient: statsservice.NewStatsServiceClient(client),
	}
}

func (s *StatsServiceClient) GetUserUplink(email string) (uint64, error) {
	return s.GetUserTraffic(fmt.Sprintf("user>>>%s>>>traffic>>>uplink", email), false)
}

func (s *StatsServiceClient) GetUserDownlink(email string) (uint64, error) {
	return s.GetUserTraffic(fmt.Sprintf("user>>>%s>>>traffic>>>downlink", email), false)
}

func (s *StatsServiceClient) GetUserTraffic(name string, reset bool) (uint64, error) {
	req := &statsservice.GetStatsRequest{
		Name:   name,
		Reset_: reset,
	}

	res, err := s.GetStats(context.Background(), req)
	if err != nil {
		if status, ok := status.FromError(err); ok && strings.HasSuffix(status.Message(), fmt.Sprintf("%s not found.", name)) {
			log.Printf("%s not found", helper.SanitizeStr(name))
			return 0, nil
		}
		log.Printf("%s", err)
		return 0, err
	}

	return uint64(res.Stat.Value), nil
}

func (s *StatsServiceClient) GetAllUserTraffic(reset bool) ([]Traffic, error) {

	regEx := `stat:{name:"(?P<tag>[\w]+)>>>(?P<name>[\w]+)>>>traffic>>>(?P<direction>[\w]+)"[\s]+value:(?P<value>[\d]+)}`
	compRegEx := regexp.MustCompile(regEx)

	middleStuff := []Traffic{}
	userTrafficExtracted := map[string]int64{}

	request := &statsservice.QueryStatsRequest{
		Pattern: "",
		Reset_:  reset,
	}

	response, err := s.QueryStats(context.Background(), request)
	if err != nil {
		log.Printf("%s", err)
		return nil, err
	}

	matches := compRegEx.FindAllStringSubmatch(response.String(), -1)

	for _, n := range matches {

		if n[1] == "user" {

			traffic, _ := strconv.ParseInt(n[4], 10, 64)

			if value, ok := userTrafficExtracted[n[2]]; ok {
				userTrafficExtracted[n[2]] = value + traffic
			} else {
				userTrafficExtracted[n[2]] = traffic
			}

		}
	}

	for name, value := range userTrafficExtracted {
		middleStuff = append(middleStuff, Traffic{
			Name:  name,
			Total: value,
		})
	}

	return middleStuff, nil
}

func NewHandlerServiceClient(client *grpc.ClientConn, inboundTag string) *HandlerServiceClient {
	return &HandlerServiceClient{
		HandlerServiceClient: command.NewHandlerServiceClient(client),
		inboundTag:           inboundTag,
	}
}

func (h *HandlerServiceClient) DelUser(email string) error {
	req := &command.AlterInboundRequest{
		Tag:       h.inboundTag,
		Operation: serial.ToTypedMessage(&command.RemoveUserOperation{Email: email}),
	}
	return h.AlterInbound(req)
}

func (h *HandlerServiceClient) AddUser(u User) error {

	user := &protocol.User{
		Level: LEVEL,
		Email: u.Email_As_Id,
		Account: serial.ToTypedMessage(&vmess.Account{
			Id:               u.UUID,
			AlterId:          ALTERID,
			SecuritySettings: &protocol.SecurityConfig{Type: protocol.SecurityType_AUTO},
		}),
	}
	req := &command.AlterInboundRequest{
		Tag:       h.inboundTag,
		Operation: serial.ToTypedMessage(&command.AddUserOperation{User: user}),
	}
	return h.AlterInbound(req)
}

func (h *HandlerServiceClient) AlterInbound(req *command.AlterInboundRequest) error {

	_, err := h.HandlerServiceClient.AlterInbound(context.Background(), req)
	return err

}
