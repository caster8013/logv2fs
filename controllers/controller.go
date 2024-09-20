package controllers

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"log"
	"os"
	"strconv"

	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	uuid "github.com/nu7hatch/gouuid"
	"gopkg.in/yaml.v2"

	"github.com/xvv6u577/logv2fs/database"

	helper "github.com/xvv6u577/logv2fs/helpers"

	"github.com/xvv6u577/logv2fs/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var (
	// userCollection     *mongo.Collection = database.OpenCollection(database.Client, "USERS")
	// nodeCollection     *mongo.Collection = database.OpenCollection(database.Client, "NODES")
	globalCollection     *mongo.Collection = database.OpenCollection(database.Client, "GLOBAL")
	MoniteringDomainsCol *mongo.Collection = database.OpenCollection(database.Client, "Monitering_Domains")
	nodeTrafficLogsCol                     = database.OpenCollection(database.Client, "NODE_TRAFFIC_LOGS")
	userTrafficLogsCol                     = database.OpenCollection(database.Client, "USER_TRAFFIC_LOGS")
	validate                               = validator.New()
	CURRENT_DOMAIN                         = os.Getenv("CURRENT_DOMAIN")
	CREDIT                                 = os.Getenv("CREDIT")
	PUBLIC_KEY                             = os.Getenv("PUBLIC_KEY")
	SHORT_ID                               = os.Getenv("SHORT_ID")
)

type (
	User            = model.User
	TrafficAtPeriod = model.TrafficAtPeriod
	Node            = model.Node
	Domain          = model.Domain
	SingboxYAML     = model.SingboxYAML
	SingboxJSON     = model.SingboxJSON
	RealityJSON     = model.RealityJSON
	Hysteria2JSON   = model.Hysteria2JSON
	RealityYAML     = model.RealityYAML
	Hysteria2YAML   = model.Hysteria2YAML
	CFVlessJSON     = model.CFVlessJSON
	CFVlessYAML     = model.CFVlessYAML
	UserTrafficLogs = model.UserTrafficLogs
	NodeTrafficLogs = model.NodeTrafficLogs
)

// HashPassword is used to encrypt the password before it is stored in the DB
func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}
	return string(bytes)
}

// VerifyPassword checks the input password while verifying it with the passward in the DB.
func VerifyPassword(userPassword string, providedPassword string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(providedPassword), []byte(userPassword))
	check := true
	msg := ""

	if err != nil {
		msg = "login or passowrd is incorrect"
		check = false
	}

	return check, msg
}

// Renews the user tokens when they login
func UpdateAllTokens(signedToken string, signedRefreshToken string, userId string) {

	var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
	var updateObj primitive.D

	updateObj = append(updateObj, bson.E{Key: "token", Value: signedToken})
	updateObj = append(updateObj, bson.E{Key: "refresh_token", Value: signedRefreshToken})

	Updated_at := time.Now()
	updateObj = append(updateObj, bson.E{Key: "updated_at", Value: Updated_at})

	upsert := true
	filter := bson.M{"user_id": userId}
	opt := options.UpdateOptions{
		Upsert: &upsert,
	}
	_, err := userTrafficLogsCol.UpdateOne(
		ctx,
		filter,
		bson.D{{Key: "$set", Value: updateObj}},
		&opt,
	)
	defer cancel()

	if err != nil {
		log.Printf("Error: %s", err.Error())
	}

}

// check if a string in a slice
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// CreateUser is the api used to tget a single user
func SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {

		if err := helper.CheckUserType(c, "admin"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user UserTrafficLogs
		var current = time.Now()

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("BindJSON error: %v", err)
			return
		}

		validationErr := validate.Struct(user)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			log.Printf("validate error: %v", validationErr)
			return
		}

		user_email := helper.SanitizeStr(user.Email_As_Id)
		count, err := userTrafficLogsCol.CountDocuments(context.TODO(), bson.M{"email_as_id": user_email})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while checking for the email"})
			log.Printf("Checking email error: %s", err.Error())
			return
		}

		if count > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "this email already exists"})
			log.Printf("this email already exists")
			return
		}

		if user.Name == "" {
			user.Name = user_email
		}

		password := HashPassword(user_email)
		user.Password = password

		user.CreatedAt = current
		user.UpdatedAt = current

		uuidV4, _ := uuid.NewV4()
		user.UUID = uuidV4.String()

		user_role := "plain"
		user.Used = 0

		if user.Credit == 0 {
			credit, _ := strconv.ParseInt(CREDIT, 10, 64)
			user.Credit = credit
		}

		user.ID = primitive.NewObjectID()
		user.User_id = user.ID.Hex()
		token, refreshToken, _ := helper.GenerateAllTokens(user_email, user.UUID, user.Name, user_role, user.User_id)
		user.Token = &token
		user.Refresh_token = &refreshToken

		user.HourlyLogs = []struct {
			Timestamp time.Time `json:"timestamp" bson:"timestamp"`
			Traffic   int64     `json:"traffic" bson:"traffic"`
		}{}
		user.DailyLogs = []struct {
			Date    string `json:"date" bson:"date"`
			Traffic int64  `json:"traffic" bson:"traffic"`
		}{}
		user.MonthlyLogs = []struct {
			Month   string `json:"month" bson:"month"`
			Traffic int64  `json:"traffic" bson:"traffic"`
		}{}
		user.YearlyLogs = []struct {
			Year    string `json:"year" bson:"year"`
			Traffic int64  `json:"traffic" bson:"traffic"`
		}{}

		_, err = userTrafficLogsCol.InsertOne(context.Background(), user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("error occured while inserting user traffic logs: %v", err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "user " + user.Name + " created successfully"})
	}
}

// Login is the api used to get a single user
func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		var boundUser, foundUser, finalUser UserTrafficLogs

		if err := c.BindJSON(&boundUser); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		sanitized_email := helper.SanitizeStr(boundUser.Email_As_Id)
		err := userTrafficLogsCol.FindOne(ctx, bson.M{"email_as_id": sanitized_email}).Decode(&foundUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		passwordIsValid, msg := VerifyPassword(boundUser.Password, foundUser.Password)
		if !passwordIsValid {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			log.Printf("password is not valid: %s", msg)
			return
		}

		token, refreshToken, _ := helper.GenerateAllTokens(sanitized_email, foundUser.UUID, foundUser.Name, foundUser.Role, foundUser.User_id)

		UpdateAllTokens(token, refreshToken, foundUser.User_id)
		var projections = bson.D{
			{Key: "token", Value: 1},
		}

		err = userTrafficLogsCol.FindOne(ctx, bson.M{"email_as_id": sanitized_email}, options.FindOne().SetProjection(projections)).Decode(&finalUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		c.JSON(http.StatusOK, finalUser)
	}
}

func EditUser() gin.HandlerFunc {
	return func(c *gin.Context) {

		if err := helper.CheckUserType(c, "admin"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var user, foundUser UserTrafficLogs

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		err := userTrafficLogsCol.FindOne(ctx, bson.M{"email_as_id": helper.SanitizeStr(user.Email_As_Id)}).Decode(&foundUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "email is incorrect"})
			log.Printf("email is incorrect")
			return
		}

		var replacedDocument bson.M
		newFoundUser := bson.M{}

		if foundUser.Role != user.Role {
			newFoundUser["role"] = user.Role
		}

		if user.Password != "" {
			password := HashPassword(user.Password)
			if foundUser.Password != user.Password && foundUser.Password != password {
				newFoundUser["password"] = password
			}
		}
		if foundUser.Name != user.Name {
			newFoundUser["name"] = user.Name
		}

		if len(newFoundUser) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no new value in post data."})
			log.Printf("no new value in post data.")
			return
		}

		err = userTrafficLogsCol.FindOneAndUpdate(
			ctx,
			bson.M{"email_as_id": helper.SanitizeStr(user.Email_As_Id)},
			bson.M{"$set": newFoundUser},
			options.FindOneAndUpdate().SetUpsert(false),
		).Decode(&replacedDocument)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		c.JSON(http.StatusOK, foundUser)
	}
}

func OfflineUserByName() gin.HandlerFunc {
	return func(c *gin.Context) {

		if err := helper.CheckUserType(c, "admin"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		name := c.Param("name")

		filter := bson.D{primitive.E{Key: "email_as_id", Value: name}}
		update := bson.M{"$set": bson.M{"status": "deleted"}}
		_, err := userTrafficLogsCol.UpdateOne(ctx, filter, update)
		if err != nil {
			msg := "database user info update failed."
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			log.Printf("%s", msg)
			return
		}

		log.Printf("Take user %s offline successfully!", name)
		c.JSON(http.StatusOK, gin.H{"message": "Take user " + name + " offline successfully!"})
	}

}

func OnlineUserByName() gin.HandlerFunc {
	return func(c *gin.Context) {

		if err := helper.CheckUserType(c, "admin"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		name := c.Param("name")

		var user UserTrafficLogs
		filter := bson.M{"email_as_id": name}
		var projections = bson.D{
			{Key: "email_as_id", Value: 1},
			{Key: "name", Value: 1},
		}
		err := userTrafficLogsCol.FindOne(context.TODO(), filter, options.FindOne().SetProjection(projections)).Decode(&user)
		if err != nil {
			msg := "database get user failed."
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			log.Printf("%s", msg)
			return
		}

		update := bson.M{"$set": bson.M{"status": "plain"}}
		_, err = userTrafficLogsCol.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			msg := "database user info update failed."
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			log.Printf("%s", msg)
			return
		}

		log.Printf("Take user %s online successfully!", user.Name)
		c.JSON(http.StatusOK, gin.H{"message": "Take user " + user.Name + " online successfully!"})
	}

}

func DeleteUserByUserName() gin.HandlerFunc {
	return func(c *gin.Context) {

		if err := helper.CheckUserType(c, "admin"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		name := c.Param("name")

		var user UserTrafficLogs
		filter := bson.M{"email_as_id": name}
		var projections = bson.D{
			{Key: "email_as_id", Value: 1},
			{Key: "name", Value: 1},
		}
		err := userTrafficLogsCol.FindOne(context.TODO(), filter, options.FindOne().SetProjection(projections)).Decode(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("DeleteUserByUserName: %s", err.Error())
			return
		}

		// delete user from userTrafficLogsCol
		_, err = userTrafficLogsCol.DeleteOne(context.TODO(), filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("DeleteUserByUserName: %s", err.Error())
			return
		}

		log.Printf("Delete user %s successfully!", user.Name)
		c.JSON(http.StatusOK, gin.H{"message": "Delete user " + user.Name + " successfully!"})
	}
}

func GetAllUsers() gin.HandlerFunc {
	return func(c *gin.Context) {

		if err := helper.CheckUserType(c, "admin"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		pipeline := mongo.Pipeline{
			{{Key: "$project", Value: bson.D{
				{Key: "email_as_id", Value: 1},
				{Key: "uuid", Value: 1},
				{Key: "name", Value: 1},
				{Key: "role", Value: 1},
				{Key: "status", Value: 1},
				{Key: "used", Value: 1},
				{Key: "daily_logs", Value: bson.D{
					{Key: "$slice", Value: bson.A{
						bson.D{
							{Key: "$sortArray", Value: bson.D{
								{Key: "input", Value: "$daily_logs"},
								{Key: "sortBy", Value: bson.D{{Key: "date", Value: -1}}},
							}},
						},
						10,
					}},
				}},
				{Key: "monthly_logs", Value: bson.D{
					{Key: "$slice", Value: bson.A{
						bson.D{
							{Key: "$sortArray", Value: bson.D{
								{Key: "input", Value: "$monthly_logs"},
								{Key: "sortBy", Value: bson.D{{Key: "month", Value: -1}}},
							}},
						},
						10,
					}},
				}},
				{Key: "yearly_logs", Value: bson.D{
					{Key: "$slice", Value: bson.A{
						bson.D{
							{Key: "$sortArray", Value: bson.D{
								{Key: "input", Value: "$yearly_logs"},
								{Key: "sortBy", Value: bson.D{{Key: "year", Value: -1}}},
							}},
						},
						10,
					}},
				}},
			}}},
		}

		cursor, err := userTrafficLogsCol.Aggregate(ctx, pipeline)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("GetAllUsers: %s", err.Error())
			return
		}
		defer cursor.Close(ctx)

		var results []UserTrafficLogs
		if err = cursor.All(ctx, &results); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("GetAllUsers: %s", err.Error())
			return
		}

		c.JSON(http.StatusOK, results)

	}
}

func GetUserByName() gin.HandlerFunc {
	return func(c *gin.Context) {

		name := c.Param("name")

		if err := helper.MatchUserTypeAndName(c, name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("GetUserByName: %s", err.Error())
			return
		}

		var projections = bson.D{
			{Key: "email_as_id", Value: 1},
			{Key: "used", Value: 1},
			{Key: "uuid", Value: 1},
			{Key: "name", Value: 1},
			{Key: "status", Value: 1},
			{Key: "role", Value: 1},
			{Key: "credit", Value: 1},
			{Key: "daily_logs", Value: 1},
			{Key: "monthly_logs", Value: 1},
			{Key: "yearly_logs", Value: 1},
		}

		var user UserTrafficLogs
		err := userTrafficLogsCol.FindOne(context.Background(), bson.M{"email_as_id": name}, options.FindOne().SetProjection(projections)).Decode(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("GetUserByName: %s", err.Error())
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

func WriteToDB() gin.HandlerFunc {
	return func(c *gin.Context) {

		if err := helper.CheckUserType(c, "admin"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// err := localCron.Log_basicAction()
		// if err != nil {
		// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		// 	log.Printf("Write to DB failed: %s", err.Error())
		// 	return
		// }

		log.Println("Write to DB by hand!")
		c.JSON(http.StatusOK, gin.H{"message": "Write to DB successfully!"})
	}
}

func GetSubscripionURL() gin.HandlerFunc {
	return func(c *gin.Context) {

		var subscription []byte
		var err error
		name := helper.SanitizeStr(c.Param("name"))

		// get globalVariable from GlobelCollection ActiveGlobalNodes
		var activeGlobalNodes []Domain

		cur, err := globalCollection.Find(context.TODO(), bson.D{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while getting globalVariable"})
			log.Printf("Getting globalVariable error: %s", err.Error())
			return
		}
		defer cur.Close(context.Background())

		cur.All(context.Background(), &activeGlobalNodes)

		// projections include status, user_id, uuid,
		var projections = bson.D{
			{Key: "status", Value: 1},
			{Key: "user_id", Value: 1},
			{Key: "uuid", Value: 1},
		}
		var user UserTrafficLogs
		err = userTrafficLogsCol.FindOne(context.TODO(), bson.M{"email_as_id": name}, options.FindOne().SetProjection(projections)).Decode(&user)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("GetSubscripionURL error: %v", err)
			return
		}

		if user.Status == "plain" {
			var sub string
			for _, node := range activeGlobalNodes {

				if node.Type == "reality" {
					if len(sub) == 0 {
						sub = "vless://" + user.UUID + "@" + node.IP + ":" + node.SERVER_PORT + "?encryption=none&flow=xtls-rprx-vision&security=reality&sni=itunes.apple.com&fp=chrome&pbk=" + node.PUBLIC_KEY + "&sid=" + node.SHORT_ID + "&type=tcp&headerType=none#" + node.Remark
					} else {
						sub = sub + "\n" + "vless://" + user.UUID + "@" + node.IP + ":" + node.SERVER_PORT + "?encryption=none&flow=xtls-rprx-vision&security=reality&sni=itunes.apple.com&fp=chrome&pbk=" + node.PUBLIC_KEY + "&sid=" + node.SHORT_ID + "&type=tcp&headerType=none#" + node.Remark
					}
				}

				if node.Type == "hysteria2" {
					if len(sub) == 0 {
						sub = "hysteria2://" + user.User_id + "@" + node.IP + ":" + node.SERVER_PORT + "?insecure=1&sni=bing.com#" + node.Remark
					} else {
						sub = sub + "\n" + "hysteria2://" + user.User_id + "@" + node.IP + ":" + node.SERVER_PORT + "?insecure=1&sni=bing.com#" + node.Remark
					}
				}

				if node.Type == "vlessCDN" {
					if len(sub) == 0 {
						sub = "vless://" + node.UUID + "@" + node.IP + ":" + node.SERVER_PORT + "?encryption=none&security=tls&sni=" + node.Domain + "&fp=randomized&type=ws&host=" + node.Domain + "&path=%2F%3Fed%3D2048#" + node.Remark
					} else {
						sub = sub + "\n" + "vless://" + node.UUID + "@" + node.IP + ":" + node.SERVER_PORT + "?encryption=none&security=tls&sni=" + node.Domain + "&fp=randomized&type=ws&host=" + node.Domain + "&path=%2F%3Fed%3D2048#" + node.Remark
					}
				}
			}

			subscription = []byte(b64.StdEncoding.EncodeToString([]byte(sub)))
		} else {
			subscription, err = os.ReadFile(helper.CurrentPath() + "/config/error.txt")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				log.Printf("GetSubscripionURL error: %v", err)
				// return
			}
		}

		c.Data(http.StatusOK, "text/plain", subscription)
	}
}

// ReturnSingboxJson
func ReturnSingboxJson() gin.HandlerFunc {
	return func(c *gin.Context) {

		name := helper.SanitizeStr(c.Param("name"))

		var err error
		var jsonFile []byte
		var singboxJSON = SingboxJSON{}
		var user UserTrafficLogs

		var projections = bson.D{
			{Key: "status", Value: 1},
			{Key: "user_id", Value: 1},
			{Key: "uuid", Value: 1},
		}
		err = userTrafficLogsCol.FindOne(context.TODO(), bson.M{"email_as_id": name}, options.FindOne().SetProjection(projections)).Decode(&user)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("ReturnSingboxJson failed: %s", err.Error())
			return
		}

		// get globalVariable from GlobelCollection ActiveGlobalNodes
		var activeGlobalNodes []Domain

		cur, err := globalCollection.Find(context.TODO(), bson.D{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while getting globalVariable"})
			log.Printf("Getting globalVariable error: %s", err.Error())
			return
		}
		defer cur.Close(context.Background())

		cur.All(context.Background(), &activeGlobalNodes)

		if user.Status == "plain" {

			jsonFile, err = os.ReadFile(helper.CurrentPath() + "/config/template_singbox.json")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				log.Printf("error: %v", err)
				return
			}

			err = json.Unmarshal(jsonFile, &singboxJSON)
			if err != nil {
				log.Printf("error: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// append reality and hysteria2 nodes to outbounds in jsonfile.
			for _, node := range activeGlobalNodes {

				server_port, _ := strconv.Atoi(node.SERVER_PORT)
				var outboundTags = []string{
					"manual-select",
					"auto",
					"WeChat",
					"Apple",
					"Microsoft",
				}

				if node.Type == "reality" {

					for i, outbound := range singboxJSON.Outbounds {
						if outboundMap, ok := outbound.(map[string]interface{}); ok {
							if Contains(outboundTags, outboundMap["tag"].(string)) || (node.EnableOpenai) && outboundMap["tag"] == "Openai" {
								if outbounds, ok := singboxJSON.Outbounds[i].(map[string]interface{}); ok {
									if outboundsList, ok := outbounds["outbounds"].([]interface{}); ok {
										singboxJSON.Outbounds[i].(map[string]interface{})["outbounds"] = append(outboundsList, node.Remark)
									}
								}
							}
						}
					}

					singboxJSON.Outbounds = append(singboxJSON.Outbounds, RealityJSON{
						Tag:            node.Remark,
						Type:           "vless",
						UUID:           user.UUID,
						ServerPort:     server_port,
						Flow:           "xtls-rprx-vision",
						PacketEncoding: "xudp",
						Server:         node.IP,
						TLS: struct {
							Enabled    bool   `json:"enabled"`
							ServerName string `json:"server_name"`
							Utls       struct {
								Enabled     bool   `json:"enabled"`
								Fingerprint string `json:"fingerprint"`
							} `json:"utls"`
							Reality struct {
								Enabled   bool   `json:"enabled"`
								PublicKey string `json:"public_key"`
								ShortID   string `json:"short_id"`
							} `json:"reality"`
						}{
							Enabled:    true,
							ServerName: "itunes.apple.com",
							Utls: struct {
								Enabled     bool   `json:"enabled"`
								Fingerprint string `json:"fingerprint"`
							}{
								Enabled:     true,
								Fingerprint: "chrome",
							},
							Reality: struct {
								Enabled   bool   `json:"enabled"`
								PublicKey string `json:"public_key"`
								ShortID   string `json:"short_id"`
							}{
								Enabled:   true,
								PublicKey: node.PUBLIC_KEY,
								ShortID:   node.SHORT_ID,
							},
						},
					})
				}

				if node.Type == "hysteria2" {

					for i, outbound := range singboxJSON.Outbounds {
						if outboundMap, ok := outbound.(map[string]interface{}); ok {
							if Contains(outboundTags, outboundMap["tag"].(string)) || (node.EnableOpenai) && outboundMap["tag"] == "Openai" {
								if outbounds, ok := singboxJSON.Outbounds[i].(map[string]interface{}); ok {
									if outboundsList, ok := outbounds["outbounds"].([]interface{}); ok {
										singboxJSON.Outbounds[i].(map[string]interface{})["outbounds"] = append(outboundsList, node.Remark)
									}
								}
							}
						}
					}

					singboxJSON.Outbounds = append(singboxJSON.Outbounds, Hysteria2JSON{
						Tag:        node.Remark,
						Type:       "hysteria2",
						Server:     node.IP,
						ServerPort: server_port,
						UpMbps:     100,
						DownMbps:   100,
						Password:   user.User_id,
						TLS: struct {
							Enabled    bool     `json:"enabled"`
							ServerName string   `json:"server_name"`
							Insecure   bool     `json:"insecure"`
							Alpn       []string `json:"alpn"`
						}{
							Enabled:    true,
							ServerName: "bing.com",
							Insecure:   true,
							Alpn:       []string{"h3"},
						},
					})
				}

				if node.Type == "vlessCDN" {

					for i, outbound := range singboxJSON.Outbounds {
						if outboundMap, ok := outbound.(map[string]interface{}); ok {
							if Contains(outboundTags, outboundMap["tag"].(string)) {
								if outbounds, ok := singboxJSON.Outbounds[i].(map[string]interface{}); ok {
									if outboundsList, ok := outbounds["outbounds"].([]interface{}); ok {
										singboxJSON.Outbounds[i].(map[string]interface{})["outbounds"] = append(outboundsList, node.Remark)
									}
								}
							}
						}
					}

					singboxJSON.Outbounds = append(singboxJSON.Outbounds, CFVlessJSON{
						Tag:        node.Remark,
						Type:       "vless",
						Server:     node.IP,
						ServerPort: server_port,
						UUID:       node.UUID,
						Flow:       "",
						TLS: struct {
							Enabled    bool   `json:"enabled"`
							ServerName string `json:"server_name"`
							Insecure   bool   `json:"insecure"`
							Utls       struct {
								Enabled     bool   `json:"enabled"`
								Fingerprint string `json:"fingerprint"`
							} `json:"utls"`
						}{
							Enabled:    true,
							ServerName: node.Domain,
							Insecure:   false,
							Utls: struct {
								Enabled     bool   `json:"enabled"`
								Fingerprint string `json:"fingerprint"`
							}{
								Enabled:     true,
								Fingerprint: "chrome",
							},
						},
						Multiplex: struct {
							Enabled    bool   `json:"enabled"`
							Protocol   string `json:"protocol"`
							MaxStreams int    `json:"max_streams"`
						}{
							Enabled:    false,
							Protocol:   "smux",
							MaxStreams: 32,
						},
						PacketEncoding: "xudp",
						Transport: struct {
							Type    string `json:"type"`
							Path    string `json:"path"`
							Headers struct {
								Host string `json:"Host"`
							} `json:"headers"`
						}{
							Type: "ws",
							Path: "/?ed=2048",
							Headers: struct {
								Host string `json:"Host"`
							}{
								Host: node.Domain,
							},
						},
					})

				}
			}

		} else {
			jsonFile, err = os.ReadFile(helper.CurrentPath() + "/config/error.json")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				log.Printf("error: %v", err)
				return
			}

			err = json.Unmarshal(jsonFile, &singboxJSON)
			if err != nil {
				log.Printf("error: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		c.JSON(http.StatusOK, singboxJSON)
	}
}

// ReturnVergeYAML: return yaml file
func ReturnVergeYAML() gin.HandlerFunc {
	return func(c *gin.Context) {

		name := helper.SanitizeStr(c.Param("name"))

		var err error
		var yamlFile []byte
		var singboxYAML = SingboxYAML{}

		var projections = bson.D{
			{Key: "status", Value: 1},
			{Key: "user_id", Value: 1},
			{Key: "uuid", Value: 1},
		}
		var user UserTrafficLogs
		err = userTrafficLogsCol.FindOne(context.TODO(), bson.M{"email_as_id": name}, options.FindOne().SetProjection(projections)).Decode(&user)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("ReturnVergeYAML failed: %s", err.Error())
			return
		}

		// get globalVariable from GlobelCollection ActiveGlobalNodes
		var activeGlobalNodes []Domain
		cur, err := globalCollection.Find(context.TODO(), bson.D{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while getting globalVariable"})
			log.Printf("Getting globalVariable error: %s", err.Error())
			return
		}
		defer cur.Close(context.Background())

		cur.All(context.Background(), &activeGlobalNodes)

		if user.Status == "plain" {
			yamlFile, err = os.ReadFile(helper.CurrentPath() + "/config/template_verge.yaml")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				log.Printf("error: %v", err)
				return
			}

			err = yaml.Unmarshal(yamlFile, &singboxYAML)
			if err != nil {
				log.Printf("error: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// append reality and hysteria2 nodes to outbounds in yamlfile.
			for _, node := range activeGlobalNodes {

				server_port, _ := strconv.Atoi(node.SERVER_PORT)
				if node.Type == "reality" {

					for i, proxy := range singboxYAML.ProxyGroups {
						if proxy.Type == "select" || proxy.Type == "url-test" {
							singboxYAML.ProxyGroups[i].Proxies = append(singboxYAML.ProxyGroups[i].Proxies, node.Remark)
						}
					}

					singboxYAML.Proxies = append(singboxYAML.Proxies, RealityYAML{
						Name:              node.Remark,
						Type:              "vless",
						Server:            node.IP,
						Port:              server_port,
						UUID:              user.UUID,
						Network:           "tcp",
						UDP:               true,
						TLS:               true,
						Flow:              "xtls-rprx-vision",
						Servername:        "itunes.apple.com",
						ClientFingerprint: "chrome",
						RealityOpts: struct {
							PublicKey string `yaml:"public-key"`
							ShortID   string `yaml:"short-id"`
						}{
							PublicKey: node.PUBLIC_KEY,
							ShortID:   node.SHORT_ID,
						},
					})
				}

				if node.Type == "hysteria2" {

					for i, proxy := range singboxYAML.ProxyGroups {
						if proxy.Type == "select" || proxy.Type == "url-test" {
							singboxYAML.ProxyGroups[i].Proxies = append(singboxYAML.ProxyGroups[i].Proxies, node.Remark)
						}
					}

					singboxYAML.Proxies = append(singboxYAML.Proxies, Hysteria2YAML{
						Name:           node.Remark,
						Type:           "hysteria2",
						Server:         node.IP,
						Port:           server_port,
						Password:       user.User_id,
						Sni:            "bing.com",
						SkipCertVerify: true,
						Alpn:           []string{"h3"},
					})
				}

				if node.Type == "vlessCDN" {

					for i, proxy := range singboxYAML.ProxyGroups {
						if proxy.Type == "select" || proxy.Type == "url-test" {
							singboxYAML.ProxyGroups[i].Proxies = append(singboxYAML.ProxyGroups[i].Proxies, node.Remark)
						}
					}

					singboxYAML.Proxies = append(singboxYAML.Proxies, CFVlessYAML{
						Name:              node.Remark,
						Type:              "vless",
						Server:            node.IP,
						Port:              server_port,
						UUID:              node.UUID,
						Network:           "ws",
						TLS:               true,
						UDP:               false,
						Servername:        node.Domain,
						ClientFingerprint: "chrome",
						WsOpts: struct {
							Path    string `yaml:"path"`
							Headers struct {
								Host string `yaml:"Host"`
							} `yaml:"headers"`
						}{
							Path: node.PATH,
							Headers: struct {
								Host string `yaml:"Host"`
							}{
								Host: node.Domain,
							},
						},
					})
				}
			}

			// if DIRECT type is not at the end of singboxYAML.ProxyGroups at select type, set it to the end.
			for i, proxy := range singboxYAML.ProxyGroups {
				if proxy.Type == "select" {
					for j, p := range proxy.Proxies {
						if p == "DIRECT" {
							singboxYAML.ProxyGroups[i].Proxies = append(singboxYAML.ProxyGroups[i].Proxies[:j], singboxYAML.ProxyGroups[i].Proxies[j+1:]...)
							singboxYAML.ProxyGroups[i].Proxies = append(singboxYAML.ProxyGroups[i].Proxies, "DIRECT")
						}
					}
				}
			}

		} else {
			yamlFile, err = os.ReadFile(helper.CurrentPath() + "/config/error.yaml")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				log.Printf("error: %v", err)
				return
			}

			err = yaml.Unmarshal(yamlFile, &singboxYAML)
			if err != nil {
				log.Fatalf("error: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		c.YAML(http.StatusOK, singboxYAML)
	}
}
