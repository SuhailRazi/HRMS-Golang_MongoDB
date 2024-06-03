package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbName = "fiber-hrms"
const mongoURI = "mongodb://localhost:27017/" + dbName

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

func Connect() error {

	// Connect to database
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}

	mg = MongoInstance{
		Client: client,
		Db:     db,
	}
	return nil
}

func main() {

	if err := Connect(); err != nil {
		log.Fatal("mongo error ", err)
	}
	app := fiber.New()

	app.Get("/employee", func(c *fiber.Ctx) error {

		query := bson.D{{}}
		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		var employees []Employee = make([]Employee, 0)

		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.JSON(employees)
	})
	app.Post("/employee", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		employee.ID = ""

		insertionResult, err := collection.InsertOne(c.Context(), employee)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}

		createdRecord := collection.FindOne(c.Context(), filter)

		createdEmployee := &Employee{}
		createdRecord.Decode(createdEmployee)

		return c.Status(201).JSON(createdEmployee)
	})
	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		idParams := c.Params("id")
		collection := mg.Db.Collection("employees")

		empId, err := primitive.ObjectIDFromHex(idParams)
		if err != nil {
			return c.SendStatus(400)
		}

		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: empId}}
		update := bson.D{}

		// Only update fields that are non zero
		if employee.Name != "" {
			update = append(update, bson.E{Key: "name", Value: employee.Name})
		}

		if employee.Age != 0 {
			update = append(update, bson.E{Key: "age", Value: employee.Age})
		}

		if employee.Salary != 0 {
			update = append(update, bson.E{Key: "salary", Value: employee.Salary})
		}

		if len(update) == 0 {
			return c.SendStatus(400)
		}

		updateQuery := bson.D{{Key: "$set", Value: update}}

		err = collection.FindOneAndUpdate(c.Context(), query, updateQuery).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(400)
			}
			return c.SendStatus(500)
		}

		employee.ID = idParams

		return c.Status(200).JSON(employee)

	})
	app.Delete("/employee/:id", func(c *fiber.Ctx) error {

		collection := mg.Db.Collection("employees")

		// checking and assigning in 1 step
		empId, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			return c.SendStatus(400)
		}

		query := bson.D{{Key: "_id", Value: empId}}

		result, err := collection.DeleteOne(c.Context(), &query)

		if err != nil {
			return c.SendStatus(500)
		}

		if result.DeletedCount < 1 {
			return c.SendStatus(404)
		}

		return c.Status(200).JSON("Record Deleted")

	})

	// start the server
	log.Fatal(app.Listen(":3000"))
}
