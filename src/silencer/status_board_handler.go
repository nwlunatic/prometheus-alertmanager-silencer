package silencer

import (
	"net/http"
)

type StatusBoardHandler struct {
	statusBoard *StatusBoard
}

func NewStatusBoardHandler(
	statusBoard *StatusBoard,
) *StatusBoardHandler {
	return &StatusBoardHandler{
		statusBoard,
	}
}

func (h *StatusBoardHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statusBoard, err := h.statusBoard.Render()
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		}

		_, _ = w.Write(statusBoard)
	}
}
