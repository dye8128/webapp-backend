package handler

import (
	"database/sql"
	"errors"

	// "hash"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	db *sqlx.DB
}

func NewHandler(db *sqlx.DB) *Handler {
	return &Handler{db: db}
}

type City struct {
	ID          int            `json:"id,omitempty"  db:"ID"`
	Name        sql.NullString `json:"name,omitempty"  db:"Name"`
	CountryCode sql.NullString `json:"countryCode,omitempty"  db:"CountryCode"`
	District    sql.NullString `json:"district,omitempty"  db:"District"`
	Population  sql.NullInt64  `json:"population,omitempty"  db:"Population"`
}

type Country struct {
	Code  string `json:"code,omitempty"  db:"Code"`
	Name  string `json:"name,omitempty"  db:"Name"`
}

type LoginRequestBody struct {
	Username string `json:"username,omitempty" form:"username"`
	Password string `json:"password,omitempty" form:"password"`
}

type User struct {
	Username   string `json:"username,omitempty"  db:"Username"`
	HashedPass string `json:"-"  db:"HashedPass"`
}

type Me struct {
	Username string `json:"username,omitempty"  db:"username"`
}

func (h *Handler) GetCityInfoHandler(c echo.Context) error {
	cityName := c.Param("cityName")

	var city City
	err := h.db.Get(&city, "SELECT * FROM city WHERE Name=?", cityName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.NoContent(http.StatusNotFound)
		}
		log.Printf("failed to get city data: %s\n", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, city)
}

func (h *Handler) PostCityHandler(c echo.Context) error {
	var city City
	err := c.Bind(&city)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request body")
	}

	result, err := h.db.Exec("INSERT INTO city (Name, CountryCode, District, Population) VALUES (?, ?, ?, ?)", city.Name, city.CountryCode, city.District, city.Population)
	if err != nil {
		log.Printf("failed to insert city data: %s\n", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("failed to get last insert id: %s\n", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	city.ID = int(id)

	return c.JSON(http.StatusCreated, city)
}

func (h *Handler) SignUpHandler(c echo.Context) error {
	req := LoginRequestBody{}
	err := c.Bind(&req); 
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "bad request body")
	}

	if req.Username == "" || req.Password == "" {
		return c.String(http.StatusBadRequest, "username or password is empty")
	}

	// 登録しようとしているユーザーが既にデータベース内に存在するかチェック
	var count int
	err = h.db.Get(&count, "SELECT COUNT(*) FROM users WHERE Username=?", req.Username)
	if err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	// 存在したら409 Conflictを返す
	if count > 0 {
		return c.String(http.StatusConflict, "Username is already used")
	}
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	if err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	_, err = h.db.Exec("INSERT INTO users (Username, HashedPass) VALUES (?, ?)", req.Username, hashedPass)

	if err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusCreated)
}

func (h *Handler) LoginHandler(c echo.Context) error {
	var req LoginRequestBody
	err := c.Bind(&req)
	if err != nil {
		return c.String(http.StatusBadRequest, "bad request body")
	}

	if req.Password == "" || req.Username == "" {
		return c.String(http.StatusBadRequest, "username or password is empty")
	}

	user := User{}
	err = h.db.Get(&user, "SELECT * FROM users WHERE Username=?", req.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.NoContent(http.StatusUnauthorized)
		} else {
			log.Println(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPass),[]byte(req.Password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return c.NoContent(http.StatusUnauthorized)
		} else {
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	sess, err := session.Get("sessions", c)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusInternalServerError, "something wrong in getting session")
	}
	sess.Values["userName"] = req.Username
	sess.Save(c.Request(), c.Response())

	return c.NoContent(http.StatusOK)
}

func UserAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc { 
	return func(c echo.Context) error { 
		sess, err := session.Get("sessions", c) 
		if err != nil { 
			log.Println(err) 
			return c.String(http.StatusInternalServerError, "something wrong in getting session") 
		} 
		if sess.Values["userName"] == nil { 
			return c.String(http.StatusUnauthorized, "please login") 
		} 
		c.Set("userName", sess.Values["userName"].(string)) 
		return next(c) 
	} 
} 

func GetMeHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, Me{
		Username: c.Get("userName").(string),
	})
}

func (h *Handler) GetWorldHandler(c echo.Context) error {
	data := []Country{}
	err := h.db.Select(&data, "SELECT Code, Name FROM country")
	if err != nil {
		log.Println(err)
		log.Println("failed to get country data")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, data)
}

func (h *Handler) GetCountryHandler(c echo.Context) error {
	countryCode := c.Param("countryCode")

	var data []City
	err := h.db.Select(&data, "SELECT * FROM city WHERE CountryCode=?", countryCode)
	if err != nil {
		log.Println(err)
		log.Println("failed to get city data")
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, data)
}
