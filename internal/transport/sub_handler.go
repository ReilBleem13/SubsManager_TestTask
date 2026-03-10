package transport

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ReilBleem13/internal/domain"
	ratelimiting "github.com/ReilBleem13/internal/rateLimiting"
	"github.com/ReilBleem13/internal/service"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

type SubHandler struct {
	srv    service.SubService
	logger *slog.Logger
}

func NewSubHandler(srv service.SubService, logger *slog.Logger) *SubHandler {
	return &SubHandler{
		srv:    srv,
		logger: logger,
	}
}

func (h *SubHandler) Register(mux *mux.Router, ipRateLimiter *ratelimiting.IPRateLimiter) {
	wrapDefaultMiddlewares := func(hf http.HandlerFunc) http.Handler {
		return conveyor(
			hf,
			requestIDMiddleware,
			loggingMiddleware(h.logger),
			rateLimitMiddleware(ipRateLimiter),
		)
	}

	mux.Handle("/subs", wrapDefaultMiddlewares(h.handleCreate)).Methods(http.MethodPost)
	mux.Handle("/subs", wrapDefaultMiddlewares(h.handleList)).Methods(http.MethodGet)
	mux.Handle("/subs/total", wrapDefaultMiddlewares(h.handleTotalAmount)).Methods(http.MethodGet)
	mux.Handle("/subs/{sub_id}", wrapDefaultMiddlewares(h.handleGet)).Methods(http.MethodGet)
	mux.Handle("/subs/{sub_id}", wrapDefaultMiddlewares(h.handleUpdate)).Methods(http.MethodPatch)
	mux.Handle("/subs/{sub_id}", wrapDefaultMiddlewares(h.handleDelete)).Methods(http.MethodDelete)

	mux.HandleFunc("/health", h.handleHealth).Methods(http.MethodGet)

	mux.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	)).Methods(http.MethodGet)
}

// handleHealth godoc
// @Summary      Проверка доступности сервера
// @Description  Проверяет сервер на доступность
// @Tags         health
// @Success      200
// @Router       /health [get]
func (h *SubHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

// handleCreate godoc
// @Summary      Создать подписку
// @Description  Создаёт новую подписку для пользователя
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        body body      CreateSubJSON true "Данные для создания подписки"
// @Success      201  {object}  domain.Sub
// @Failure      400  {object}  ErrorResponse "Некорректный запрос"
// @Failure      409  {object}  ErrorResponse "Объект уже существует"
// @Failure      500  {object}  ErrorResponse "Внутренняя ошибка"
// @Router       /subs [post]
func (h *SubHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var in CreateSubJSON
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode", "error", err)
		handleError(w, domain.ErrBadRequest().WithMessage(err.Error()))
		return
	}

	sub, err := h.srv.Create(r.Context(), mapCreateSubJSONToService(&in))
	if err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

// handleGet godoc
// @Summary      Получить подписку по ID
// @Description  Возвращает информацию об одной подписке
// @Tags         subscriptions
// @Produce      json
// @Param        sub_id path int true "ID подписки"
// @Success      200  {object}  domain.Sub
// @Failure      400  {object}  ErrorResponse "Некорректный запрос"
// @Failure      404  {object}  ErrorResponse "Подписка не найдена"
// @Failure      500  {object}  ErrorResponse "Внутренняя ошибка"
// @Router       /subs/{sub_id} [get]
func (h *SubHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	rawSubID := mux.Vars(r)["sub_id"]

	sub, err := h.srv.Get(r.Context(), rawSubID)
	if err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

// handleUpdate godoc
// @Summary      Обновить подписку
// @Description  Частичное обновление подписки
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        sub_id path int true "ID подписки"
// @Param        body   body UpdateSubJSON true "Поля для обновления"
// @Success      200   {object} domain.Sub
// @Failure      400  {object}  ErrorResponse "Некорректный запрос"
// @Failure      404  {object}  ErrorResponse "Подписка не найдена"
// @Failure      500  {object}  ErrorResponse "Внутренняя ошибка"
// @Router       /subs/{sub_id} [patch]
func (h *SubHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	var in UpdateSubJSON
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to decode", "error", err)
		handleError(w, domain.ErrBadRequest().WithMessage(err.Error()))
		return
	}

	rawSubID := mux.Vars(r)["sub_id"]

	sub, err := h.srv.Update(r.Context(), rawSubID, mapUpdateSubJSONToService(&in))
	if err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

// handleDelete godoc
// @Summary      Удалить подписку
// @Description  Удаляет подписку по ID
// @Tags         subscriptions
// @Produce      json
// @Param        sub_id path int true "ID подписки"
// @Success      204   "Успешно удалено"
// @Failure      404  {object}  ErrorResponse "Подписка не найдена"
// @Failure      500  {object}  ErrorResponse "Внутренняя ошибка"
// @Router       /subs/{sub_id} [delete]
func (h *SubHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	rawSubID := mux.Vars(r)["sub_id"]

	if err := h.srv.Delete(r.Context(), rawSubID); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(204)
}

// handleList godoc
// @Summary      Список подписок
// @Description  Возвращает список всех подписок
// @Tags         subscriptions
// @Produce      json
// @Param        limit     query     string   false   "Лимит объектов на страницу" example(10)
// @Param        page       query     string   false   "Номер страницы"  example(2)
// @Success      200  {array}   domain.Sub
// @Failure      500  {object}  ErrorResponse "Внутренняя ошибка"
// @Router       /subs [get]
func (h *SubHandler) handleList(w http.ResponseWriter, r *http.Request) {
	rawLimit := r.URL.Query().Get("limit")
	rawPage := r.URL.Query().Get("page")

	result, err := h.srv.List(r.Context(), rawLimit, rawPage)
	if err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleTotalAmount godoc
// @Summary      Общая сумма подписок
// @Description  Возвращает сумму стоимости подписок с возможной фильтрацией по пользователю, названию подписки и периоду.
// @Description
// @Description  Все параметры опциональные:
// @Description  - userID: фильтр по ID пользователя
// @Description  - subName: фильтр по названию
// @Description  - from / to: временной диапазон
// @Tags         subscriptions
// @Produce      json
// @Param        userID   query     string   false   "ID пользователя (строка или число)"   example(968a623d-a66d-4aec-8138-8850794fa28a)
// @Param        subName  query     string   false   "Название/тип подписки"                example("yandex")
// @Param        from     query     string   false   "Начало периода"             example(2025-01-01)
// @Param        to       query     string   false   "Конец периода"              example(2025-12-31)
// @Success      200      {object}  service.TotalAmountResponse   "Сумма и (возможно) другая статистика"
// @Failure      400  	  {object}  ErrorResponse                "Некорректный запрос"
// @Failure      500      {object}  ErrorResponse                 "Внутренняя ошибка сервера"
// @Router       /subs/total [get]
func (h *SubHandler) handleTotalAmount(w http.ResponseWriter, r *http.Request) {
	rawUserID := r.URL.Query().Get("userID")
	rawSubName := r.URL.Query().Get("subName")
	rawFrom := r.URL.Query().Get("from")
	rawTo := r.URL.Query().Get("to")

	result, err := h.srv.TotalAmount(r.Context(), &service.TotalAmountRequest{
		RawUserID:  rawUserID,
		RawSubname: rawSubName,
		RawFrom:    rawFrom,
		RawTo:      rawTo,
	})
	if err != nil {
		handleError(w, err)
		return
	}

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
