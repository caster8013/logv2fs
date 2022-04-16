package controllers

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"net/http"
	"time"

	b64 "encoding/base64"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	uuid "github.com/nu7hatch/gouuid"
	"google.golang.org/grpc"

	"github.com/caster8013/logv2rayfullstack/database"
	"github.com/caster8013/logv2rayfullstack/routine"
	"github.com/caster8013/logv2rayfullstack/v2ray"

	helper "github.com/caster8013/logv2rayfullstack/helpers"

	"github.com/caster8013/logv2rayfullstack/grpctools"
	"github.com/caster8013/logv2rayfullstack/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// var Domains = map[string]string{"w8": "w8.undervineyard.com", "rm": "rm.undervineyard.com"}
var (
	userCollection *mongo.Collection = database.OpenCollection(database.Client, "USERS")
	validate                         = validator.New()
	BOOT_MODE                        = os.Getenv("BOOT_MODE")
	V2_API_ADDRESS                   = os.Getenv("V2_API_ADDRESS")
	V2_API_PORT                      = os.Getenv("V2_API_PORT")
)

type (
	User            = model.User
	TrafficAtPeriod = model.TrafficAtPeriod
	Node            = model.Node
)

//HashPassword is used to encrypt the password before it is stored in the DB
func HashPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Panic(err)
	}

	return string(bytes)
}

//VerifyPassword checks the input password while verifying it with the passward in the DB.
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

func GenerateSubscription(user User, Domains map[string]string) string {

	var node Node
	subscription := ""

	for index, item := range Domains {

		node = Node{
			Domain:  item,
			Path:    "/" + user.Path,
			UUID:    user.UUID,
			Remark:  index,
			Version: "2",
			Port:    "443",
			Aid:     "64",
			Net:     "ws",
			Type:    "none",
			Tls:     "tls",
		}

		jsonedNode, _ := json.Marshal(node)

		if len(subscription) == 0 {
			subscription = "vmess://" + b64.StdEncoding.EncodeToString(jsonedNode)
		} else {
			subscription = subscription + "\n" + "vmess://" + b64.StdEncoding.EncodeToString(jsonedNode)
		}
	}

	return b64.StdEncoding.EncodeToString([]byte(subscription))
}

//CreateUser is the api used to tget a single user
func SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {

		if BOOT_MODE == "" {
			err := helper.CheckUserType(c, "admin")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"sign up error": err.Error()})
				log.Printf("%s", err.Error())
				return
			}
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		var user model.User
		var current = time.Now()

		CREDIT := os.Getenv("CREDIT")

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		validationErr := validate.Struct(user)
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
			log.Printf("error: %v", validationErr)
			return
		}

		count, err := userCollection.CountDocuments(ctx, bson.M{"email": user.Email})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while checking for the email"})
			log.Printf("error occured while checking for the email: %s", err.Error())
			return
		}

		if count > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "this email already exists"})
			log.Printf("this email already exists")
			return
		}

		var adminUser model.User
		userId := c.GetString("uid")
		log.Println("userId: ", userId)

		err = userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&adminUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		if user.Name == "" {
			user.Name = user.Email
		}

		if user.Path == "" {
			user.Path = "ray"
		}

		password := HashPassword(user.Password)
		user.Password = password

		user.CreatedAt = current
		user.UpdatedAt = current

		if user.UUID == "" {
			uuidV4, _ := uuid.NewV4()
			user.UUID = uuidV4.String()
		}

		user.NodeInUse = adminUser.NodeGlobal
		if user.Role == "admin" {
			user.NodeGlobal = adminUser.NodeGlobal
		}
		user.Suburl = GenerateSubscription(user, *adminUser.NodeGlobal)

		if user.Credittraffic == 0 {
			credit, _ := strconv.ParseInt(CREDIT, 10, 64)
			user.Credittraffic = credit
		}
		if user.Usedtraffic == 0 {
			user.Usedtraffic = 0
		}

		var current_year = current.Local().Format("2006")
		var current_month = current.Local().Format("200601")
		var current_day = current.Local().Format("20060102")

		user.UsedByCurrentDay = TrafficAtPeriod{
			Period: current_day,
			Amount: 0,
		}
		user.UsedByCurrentMonth = TrafficAtPeriod{
			Period: current_month,
			Amount: 0,
		}
		user.UsedByCurrentYear = TrafficAtPeriod{
			Period: current_year,
			Amount: 0,
		}

		user.TrafficByDay = []TrafficAtPeriod{}
		user.TrafficByMonth = []TrafficAtPeriod{}
		user.TrafficByYear = []TrafficAtPeriod{}

		user.ID = primitive.NewObjectID()
		user.User_id = user.ID.Hex()
		token, refreshToken, _ := helper.GenerateAllTokens(user.Email, user.UUID, user.Path, user.Role, user.User_id)
		user.Token = &token
		user.Refresh_token = &refreshToken

		var wg sync.WaitGroup
		domainsLen := len(*user.NodeInUse)
		wg.Add(domainsLen)

		for _, node := range *user.NodeInUse {

			go func(domain string) {
				defer wg.Done()
				if domain == "sl.undervineyard.com" {
					grpctools.GrpcClientToAddUser(domain, "80", user)
				} else {
					grpctools.GrpcClientToAddUser(domain, "50051", user)
				}
			}(node)

		}

		_, err = userCollection.InsertOne(ctx, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("error occured while inserting user: %v", err)
			return
		}

		err = database.Client.Database("logV2rayTrafficDB").CreateCollection(ctx, user.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("error occured while creating collection for user %s", user.Email)
			return
		}

		wg.Wait()

		fmt.Println(user.Email, "created at v2ray and database.")

		c.JSON(http.StatusOK, gin.H{"message": "user " + user.Name + " created at v2ray and database."})

	}
}

//Login is the api used to get a single user
func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		var user model.User
		var foundUser model.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		passwordIsValid, msg := VerifyPassword(user.Password, foundUser.Password)
		if !passwordIsValid {
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			log.Printf("password is not valid: %s", msg)
			return
		}

		token, refreshToken, _ := helper.GenerateAllTokens(foundUser.Email, foundUser.UUID, foundUser.Path, foundUser.Role, foundUser.User_id)

		helper.UpdateAllTokens(token, refreshToken, foundUser.User_id)
		err = userCollection.FindOne(ctx, bson.M{"user_id": foundUser.User_id}).Decode(&foundUser)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		c.JSON(http.StatusOK, foundUser)
	}
}

func GenerateConfig() gin.HandlerFunc {
	return func(c *gin.Context) {

		var name string
		if c.Query("name") != "" {
			name = c.Query("name")
		} else {
			name = c.Param("name")
		}

		user, err := database.GetUserByName(name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"email": user.Email, "uuid": user.UUID, "path": user.Path, "nodeinuse": user.NodeInUse})
	}
}

func AddNode() gin.HandlerFunc {
	return func(c *gin.Context) {

		if BOOT_MODE == "" {
			err := helper.CheckUserType(c, "admin")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"sign up error": err.Error()})
				log.Printf("%s", err.Error())
				return
			}
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		var domains *map[string]string
		var current = time.Now()

		if err := c.BindJSON(&domains); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		allUsers, err := database.GetAllUsersInfo()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		for _, user := range allUsers {
			if user.Role == "admin" {
				user.NodeGlobal = domains
			}

			user.NodeInUse = domains
			user.Suburl = GenerateSubscription(*user, *domains)
			user.UpdatedAt = current

			_, err = userCollection.ReplaceOne(ctx, bson.M{"user_id": user.User_id}, user)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				log.Printf("error: %v", err)
				return
			}

		}

		c.JSON(http.StatusOK, gin.H{"message": "node added"})
	}
}

func EditUser() gin.HandlerFunc {
	return func(c *gin.Context) {

		if BOOT_MODE == "" {
			if err := helper.CheckUserType(c, "admin"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				log.Printf("%s", err.Error())
				return
			}
		}

		var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var user, foundUser model.User

		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		err := userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
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
		if foundUser.Usedtraffic != user.Usedtraffic {
			newFoundUser["used"] = user.Usedtraffic
		}
		if foundUser.Credittraffic != user.Credittraffic {
			newFoundUser["credit"] = user.Credittraffic
		}

		if len(newFoundUser) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no new value in post data."})
			log.Printf("no new value in post data.")
			return
		}

		err = userCollection.FindOneAndUpdate(
			ctx,
			bson.M{"email": user.Email},
			bson.M{"$set": newFoundUser},
			options.FindOneAndUpdate().SetUpsert(true),
		).Decode(&replacedDocument)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		err = userCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&foundUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("error: %v", err)
			return
		}

		c.JSON(http.StatusOK, foundUser)
	}
}

func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {

		if BOOT_MODE == "" {
			if err := helper.CheckUserType(c, "admin"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				log.Printf("%s", err.Error())
				return
			}
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)

		// recordPerPage := 10
		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10
		}

		page, err1 := strconv.Atoi(c.Query("page"))
		if err1 != nil || page < 1 {
			page = 1
		}

		// startIndex := (page - 1) * recordPerPage
		startIndex, _ := strconv.Atoi(c.Query("startIndex"))

		matchStage := bson.D{{Key: "$match", Value: bson.D{{}}}}
		groupStage := bson.D{{Key: "$group", Value: bson.D{{Key: "_id", Value: bson.D{{Key: "_id", Value: "null"}}}, {Key: "total_count", Value: bson.D{{Key: "$sum", Value: 1}}}, {Key: "data", Value: bson.D{{Key: "$push", Value: "$$ROOT"}}}}}}
		projectStage := bson.D{
			{Key: "$project", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "total_count", Value: 1},
				{Key: "user_items", Value: bson.D{{Key: "$slice", Value: []interface{}{"$data", startIndex, recordPerPage}}}},
			}}}

		result, err := userCollection.Aggregate(ctx, mongo.Pipeline{matchStage, groupStage, projectStage})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while listing user items"})
			log.Printf("error occured while listing user items: %v", err)
			return
		}
		var allusers []bson.M
		if err = result.All(ctx, &allusers); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while listing user items"})
			log.Printf("error occured while listing user items: %v", err)
			return
		}
		c.JSON(http.StatusOK, allusers[0])

	}
}

//GetUser is the api used to get a single user
func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var user model.User
		userId := c.Param("user_id")

		if err := helper.MatchUserTypeAndUid(c, userId); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("%s", err.Error())
			return
		}

		err := userCollection.FindOne(ctx, bson.M{"user_id": userId}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("error occured while listing user items")
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

func TakeItOfflineByUserName() gin.HandlerFunc {
	return func(c *gin.Context) {

		if BOOT_MODE == "" {
			if err := helper.CheckUserType(c, "admin"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				log.Printf("%s", err.Error())
				return
			}
		}

		name := c.Param("name")

		user, err := database.GetUserByName(name)
		if err != nil {
			msg := "database get user failed."
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			log.Printf("%s", msg)
			return
		}

		var wg sync.WaitGroup
		domainsLen := len(*user.NodeInUse)
		wg.Add(domainsLen)

		for _, node := range *user.NodeInUse {

			go func(item string) {
				defer wg.Done()
				if item == "sl.undervineyard.com" {
					grpctools.GrpcClientToDeleteUser(item, "80", user)
				} else {
					grpctools.GrpcClientToDeleteUser(item, "50051", user)
				}
			}(node)

		}

		err = database.UpdateUserStatusByName(name, v2ray.DELETE)
		if err != nil {
			msg := "database user info update failed."
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			log.Printf("%s", msg)
			return
		}

		wg.Wait()

		c.JSON(http.StatusOK, gin.H{"message": "User " + user.Name + " is offline!"})
	}
}

func TakeItOnlineByUserName() gin.HandlerFunc {
	return func(c *gin.Context) {
		if BOOT_MODE == "" {
			if err := helper.CheckUserType(c, "admin"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				log.Printf("%s", err.Error())
				return
			}
		}

		name := c.Param("name")

		user, err := database.GetUserByName(name)
		if err != nil {
			msg := "database get user failed."
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			log.Printf("%s", msg)
			return
		}

		var wg sync.WaitGroup
		domainsLen := len(*user.NodeInUse)
		wg.Add(domainsLen)

		for _, node := range *user.NodeInUse {

			go func(item string) {
				defer wg.Done()
				if item == "sl.undervineyard.com" {
					grpctools.GrpcClientToAddUser(item, "80", user)
				} else {
					grpctools.GrpcClientToAddUser(item, "50051", user)
				}
			}(node)

		}

		err = database.UpdateUserStatusByName(name, v2ray.PLAIN)
		if err != nil {
			msg := "database user info update failed."
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			log.Printf("%s", msg)
			return
		}

		wg.Wait()

		c.JSON(http.StatusOK, gin.H{"message": "User " + user.Name + " is online!"})
	}
}

func DeleteUserByUserName() gin.HandlerFunc {
	return func(c *gin.Context) {

		if BOOT_MODE == "" {
			if err := helper.CheckUserType(c, "admin"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				log.Printf("%s", err.Error())
				return
			}
		}

		name := c.Param("name")

		user, err := database.GetUserByName(name)
		if err != nil {
			msg := "database get user failed."
			c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
			log.Printf("%s", msg)
			return
		}

		if user.Status == "plain" {

			var wg sync.WaitGroup
			domainsLen := len(*user.NodeInUse)
			wg.Add(domainsLen)

			for _, node := range *user.NodeInUse {

				go func(item string) {
					defer wg.Done()
					if item == "sl.undervineyard.com" {
						grpctools.GrpcClientToDeleteUser(item, "80", user)
					} else {
						grpctools.GrpcClientToDeleteUser(item, "50051", user)
					}
				}(node)

			}
			wg.Wait()

		}

		err = database.DeleteUserByName(name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("DeleteUserByUserName: %s", err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Delete user " + user.Name + " successfully!"})
	}
}

func GetTrafficByUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")

		if err := helper.MatchUserTypeAndName(c, name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("%s", err.Error())
			return
		}

		cmdConn, err := grpc.Dial(fmt.Sprintf("%s:%s", V2_API_ADDRESS, V2_API_PORT), grpc.WithInsecure())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("grpc dial error: %v", err)
			return
		}

		NSSClient := v2ray.NewStatsServiceClient(cmdConn)
		uplink, err := NSSClient.GetUserUplink(name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("GetUserUplink failed: %v", err)
			return
		}

		downlink, err := NSSClient.GetUserDownlink(name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("Get user %s downlink failed.", name)
			return
		}

		c.JSON(http.StatusOK, gin.H{"uplink": uplink, "downlink": downlink})
	}
}

func GetAllUserTraffic() gin.HandlerFunc {
	return func(c *gin.Context) {

		if BOOT_MODE == "" {
			if err := helper.CheckUserType(c, "admin"); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				log.Printf("%s", err.Error())
				return
			}
		}

		cmdConn, err := grpc.Dial(fmt.Sprintf("%s:%s", V2_API_ADDRESS, V2_API_PORT), grpc.WithInsecure())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("%s", err.Error())
			return
		}

		NSSClient := v2ray.NewStatsServiceClient(cmdConn)

		allTraffic, err := NSSClient.GetAllUserTraffic(false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("GetAllUserTraffic failed: %s", err.Error())
			return
		}

		c.JSON(http.StatusOK, allTraffic)
	}
}

func GetAllUsers() gin.HandlerFunc {
	return func(c *gin.Context) {

		if BOOT_MODE == "" {
			err := helper.CheckUserType(c, "admin")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				log.Printf("%s", err.Error())
				return
			}
		}

		allUsers, _ := database.GetAllUsersInfo()
		if len(allUsers) == 0 {
			c.JSON(http.StatusOK, []User{})
			return
		}
		c.JSON(http.StatusOK, allUsers)

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

		user, err := database.GetUserByName(name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("Get user by name failed: %s", err.Error())
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

func GetSubscripionURL() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")

		user, err := database.GetUserByName(name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.Printf("GetUserByName failed: %s", err.Error())
			return
		}

		c.String(http.StatusOK, user.Suburl)
	}
}

func WriteToDB() gin.HandlerFunc {
	return func(c *gin.Context) {
		if BOOT_MODE == "" {
			err := helper.CheckUserType(c, "admin")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
		err := routine.Log_basicAction()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			log.Printf("Write to DB failed: %s", err.Error())
			return
		}

		log.Println("Write to DB by hand!")
		c.JSON(http.StatusOK, gin.H{"message": "Write to DB successfully!"})
	}
}
