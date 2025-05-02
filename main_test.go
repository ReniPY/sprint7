package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCafeWhenOk(t *testing.T) {
	handler := http.HandlerFunc(mainHandle)

	requests := []string{
		"/cafe?count=2&city=moscow",
		"/cafe?city=tula",
		"/cafe?city=moscow&search=ложка",
	}
	for _, v := range requests {
		response := httptest.NewRecorder()
		req := httptest.NewRequest("GET", v, nil)

		handler.ServeHTTP(response, req)

		assert.Equal(t, http.StatusOK, response.Code)
	}
}

func TestCafeNegative(t *testing.T) {
	handler := http.HandlerFunc(mainHandle)

	requests := []struct {
		request string
		status  int
		message string
	}{
		{"/cafe", http.StatusBadRequest, "unknown city"},
		{"/cafe?city=omsk", http.StatusBadRequest, "unknown city"},
		{"/cafe?city=tula&count=na", http.StatusBadRequest, "incorrect count"},
	}
	for _, v := range requests {
		response := httptest.NewRecorder()
		req := httptest.NewRequest("GET", v.request, nil)
		handler.ServeHTTP(response, req)

		assert.Equal(t, v.status, response.Code)
		assert.Equal(t, v.message, strings.TrimSpace(response.Body.String()))
	}
}

func TestCafeCount(t *testing.T) {
	handler := http.HandlerFunc(mainHandle)

	for city := range cafeList {
		requests := []struct {
			count int
			want  int
		}{
			{0, 0},                               // Не возвращаем никаких кафе
			{1, 1},                               // Возвращаем одно кафе
			{2, 2},                               // Возвращаем два кафе
			{100, min(100, len(cafeList[city]))}, // Возвращаем все или максимум 100
		}

		for _, testCase := range requests {
			testCase := testCase // Избежание race condition
			t.Run(fmt.Sprintf("city=%s,count=%d", city, testCase.count), func(t *testing.T) {
				// Формируем запрос
				url := fmt.Sprintf("/cafe?city=%s&count=%d", city, testCase.count)
				req := httptest.NewRequest("GET", url, nil)

				// Выполняем запрос
				recorder := httptest.NewRecorder()
				handler.ServeHTTP(recorder, req)

				// Проверяем успешность выполнения запроса
				require.Equal(t, http.StatusOK, recorder.Code)

				// Читаем тело ответа и парсим его
				body := strings.TrimSpace(recorder.Body.String())
				results := strings.Split(body, ",")

				// Учтем ситуацию с пустой строкой
				length := len(results)
				if body == "" && length > 0 {
					length-- // Коррекция на случай пустой строки
				}

				// Проверяем количество полученных записей
				require.Equalf(t, testCase.want, length,
					"Ожидалось %d кафе, получено %d", testCase.want, length)
			})
		}
	}
}
func TestCafeSearch(t *testing.T) {
	handler := http.HandlerFunc(mainHandle)

	city := "moscow"

	requests := []struct {
		search    string // передаваемое значение search
		wantCount int    // ожидаемое количество кафе в ответе
	}{
		{"фасоль", 0}, // ни одно кафе не содержит слово "фасоль"
		{"кофе", 2},   // два кафе содержат слово "кофе"
		{"вилка", 1},  // одно кафе содержит слово "вилка"
	}

	for _, testCase := range requests {
		testCase := testCase // копируем экземпляр для безопасности
		t.Run(fmt.Sprintf("search=%s", testCase.search), func(t *testing.T) {
			// Формируем запрос
			url := fmt.Sprintf("/cafe?city=%s&search=%s", city, testCase.search)
			req := httptest.NewRequest("GET", url, nil)

			// Выполняем запрос
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			// Проверяем успешность выполнения запроса
			require.Equal(t, http.StatusOK, recorder.Code)

			// Читаем тело ответа и предварительная очистка
			body := strings.TrimSpace(recorder.Body.String())

			// Специальная обработка пустой строки
			if body == "" {
				assert.Equal(t, testCase.wantCount, 0)
				return
			}

			// Ищем совпадения
			results := strings.Split(body, ",")

			// Проверяем количество полученных записей
			assert.Equalf(t, testCase.wantCount, len(results),
				"Ожидалось %d записей, найдено %d", testCase.wantCount, len(results))

			// Проверяем, что каждое название содержит нужный термин
			for _, item := range results {
				assert.Containsf(t, strings.ToLower(item), strings.ToLower(testCase.search),
					"Название кафе '%s' не содержит поисковый термин '%s'", item, testCase.search)
			}
		})
	}
}
