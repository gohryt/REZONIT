package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type (
	Configuration struct {
		DSN string
	}

	Application struct {
		Pool *pgxpool.Pool
	}

	Data struct {
		ID   pgtype.UUID
		Date pgtype.Timestamp
		Data json.RawMessage
	}
)

func main() {
	c := Configuration{
		DSN: os.Getenv("DSN"),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err := Run(ctx, c)
	if err != nil {
		log.Panic(err)
	}
}

func Run(shutdown context.Context, configuration Configuration) (err error) {
	pool, err := pgxpool.New(shutdown, configuration.DSN)
	if err != nil {
		return
	}
	defer pool.Close()

	a := Application{
		Pool: pool,
	}

	fiber := fiber.New()
	defer fiber.Shutdown()

	fiber.Get("/", a.GetList)

	fiber.Post("/", a.Post)
	fiber.Delete("/", a.Delete)

	errors := make(chan error, 1)

	go func() {
		err := fiber.Listen(":3000")

		errors <- err
	}()

	select {
	case <-shutdown.Done():
		return nil
	case err = <-errors:
		return err
	}
}

func (a *Application) GetList(c *fiber.Ctx) error {
	dataList := []Data{}
	data := Data{}

	rows, err := a.Pool.Query(c.Context(), "select id, date, data from data")
	if err != nil {
		return err
	}

	for rows.Next() {
		err = rows.Scan(&data.ID, &data.Date, &data.Data)
		if err != nil {
			return err
		}

		dataList = append(dataList, data)
	}

	return c.JSON(dataList)
}

func (a *Application) Post(c *fiber.Ctx) error {
	body := c.Body()

	valid := json.Valid(body)
	if !valid {
		return fmt.Errorf("input data should be json")
	}

	data := Data{
		Data: body,
	}

	ctx := c.Context()

	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(c.Context(), "insert into data (data) values ($1) returning id, date", &data.Data).Scan(&data.ID, &data.Date)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return c.JSON(data)
}

func (a *Application) Delete(c *fiber.Ctx) error {
	data := Data{}

	ctx := c.Context()

	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(c.Context(), "delete from data where id = (select id from data order by date desc) returning id, date, data").Scan(&data.ID, &data.Date, &data.Data)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return c.JSON(data)
}
