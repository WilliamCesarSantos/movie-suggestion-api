package usecase

import (
	"context"
	"encoding/json"

	"github.com/WilliamCesarSantos/movie-suggestion/internal/domain/usecase"
	"github.com/WilliamCesarSantos/movie-suggestion/internal/infrastructure/lambda"
)

type importMoviesUseCase struct {
	importClient *lambda.ImportClient
}

func NewImportMoviesUseCase(importClient *lambda.ImportClient) usecase.ImportMoviesUseCase {
	return &importMoviesUseCase{importClient: importClient}
}

type importPayload struct {
	SearchTerms []string `json:"searchTerms"`
	MaxPages    int      `json:"maxPages"`
}

func (uc *importMoviesUseCase) Execute(ctx context.Context, searchTerms []string, maxPages int) error {
	payload := importPayload{SearchTerms: searchTerms, MaxPages: maxPages}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return uc.importClient.Invoke(ctx, data)
}
