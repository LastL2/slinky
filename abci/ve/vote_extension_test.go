package ve_test

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"cosmossdk.io/log"
	cometabci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/skip-mev/slinky/abci/preblock"
	"github.com/skip-mev/slinky/abci/strategies/codec"
	codecmocks "github.com/skip-mev/slinky/abci/strategies/codec/mocks"
	mockstrategies "github.com/skip-mev/slinky/abci/strategies/currencypair/mocks"
	"github.com/skip-mev/slinky/abci/testutils"
	slinkyabci "github.com/skip-mev/slinky/abci/types"
	"github.com/skip-mev/slinky/abci/ve"
	abcitypes "github.com/skip-mev/slinky/abci/ve/types"
	client "github.com/skip-mev/slinky/service/clients/oracle"
	"github.com/skip-mev/slinky/service/clients/oracle/mocks"
	servicemetrics "github.com/skip-mev/slinky/service/metrics"
	metricsmocks "github.com/skip-mev/slinky/service/metrics/mocks"
	servicetypes "github.com/skip-mev/slinky/service/servers/oracle/types"
	oracletypes "github.com/skip-mev/slinky/x/oracle/types"
)

var (
	btcUSD     = oracletypes.NewCurrencyPair("BTC", "USD")
	ethUSD     = oracletypes.NewCurrencyPair("ETH", "USD")
	oneHundred = big.NewInt(100)
	twoHundred = big.NewInt(200)

	nilPrices   = map[string]string{}
	singlePrice = map[string]string{
		btcUSD.String(): oneHundred.String(),
	}
	multiplePrices = map[string]string{
		btcUSD.String(): oneHundred.String(),
		ethUSD.String(): twoHundred.String(),
	}
)

type VoteExtensionTestSuite struct {
	suite.Suite
	ctx sdk.Context
}

func (s *VoteExtensionTestSuite) SetupTest() {
	s.ctx = testutils.CreateBaseSDKContext(s.T())
}

func TestVoteExtensionTestSuite(t *testing.T) {
	suite.Run(t, new(VoteExtensionTestSuite))
}

func (s *VoteExtensionTestSuite) TestExtendVoteExtension() {
	cases := []struct {
		name                 string
		oracleService        func() client.OracleClient
		currencyPairStrategy func() *mockstrategies.CurrencyPairStrategy
		expectedResponse     *abcitypes.OracleVoteExtension
		extendVoteRequest    func() *cometabci.RequestExtendVote
		expectedError        bool
	}{
		{
			name: "nil request returns an error",
			oracleService: func() client.OracleClient {
				return mocks.NewOracleClient(s.T())
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				return mockstrategies.NewCurrencyPairStrategy(s.T())
			},
			extendVoteRequest: func() *cometabci.RequestExtendVote { return nil },
		},
		{
			name: "oracle service returns no prices",
			oracleService: func() client.OracleClient {
				mockServer := mocks.NewOracleClient(s.T())

				mockServer.On("Prices", mock.Anything, mock.Anything).Return(
					&servicetypes.QueryPricesResponse{
						Prices: nilPrices,
					},
					nil,
				)

				return mockServer
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				return mockstrategies.NewCurrencyPairStrategy(s.T())
			},
			expectedResponse: &abcitypes.OracleVoteExtension{
				Prices: nil,
			},
		},
		{
			name: "oracle service returns a single price",
			oracleService: func() client.OracleClient {
				mockServer := mocks.NewOracleClient(s.T())

				mockServer.On("Prices", mock.Anything, mock.Anything).Return(
					&servicetypes.QueryPricesResponse{
						Prices: singlePrice,
					},
					nil,
				)

				return mockServer
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				cps := mockstrategies.NewCurrencyPairStrategy(s.T())

				cps.On("ID", mock.Anything, btcUSD).Return(uint64(0), nil)
				cps.On("GetEncodedPrice", mock.Anything, btcUSD, oneHundred).Return(oneHundred.Bytes(), nil)

				return cps
			},
			expectedResponse: &abcitypes.OracleVoteExtension{
				Prices: map[uint64][]byte{
					0: oneHundred.Bytes(),
				},
			},
		},
		{
			name: "oracle service returns multiple prices",
			oracleService: func() client.OracleClient {
				mockServer := mocks.NewOracleClient(s.T())

				mockServer.On("Prices", mock.Anything, mock.Anything).Return(
					&servicetypes.QueryPricesResponse{
						Prices: multiplePrices,
					},
					nil,
				)

				return mockServer
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				cps := mockstrategies.NewCurrencyPairStrategy(s.T())

				cps.On("ID", mock.Anything, btcUSD).Return(uint64(0), nil)
				cps.On("GetEncodedPrice", mock.Anything, btcUSD, oneHundred).Return(oneHundred.Bytes(), nil)

				cps.On("ID", mock.Anything, ethUSD).Return(uint64(1), nil)
				cps.On("GetEncodedPrice", mock.Anything, ethUSD, twoHundred).Return(twoHundred.Bytes(), nil)

				return cps
			},
			expectedResponse: &abcitypes.OracleVoteExtension{
				Prices: map[uint64][]byte{
					0: oneHundred.Bytes(),
					1: twoHundred.Bytes(),
				},
			},
		},
		{
			name: "oracle service panics",
			oracleService: func() client.OracleClient {
				mockServer := mocks.NewOracleClient(s.T())

				mockServer.On("Prices", mock.Anything, mock.Anything).Panic("panic")

				return mockServer
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				return mockstrategies.NewCurrencyPairStrategy(s.T())
			},
			expectedResponse: &abcitypes.OracleVoteExtension{
				Prices: nil,
			},
			expectedError: true,
		},
		{
			name: "oracle service returns an nil response",
			oracleService: func() client.OracleClient {
				mockServer := mocks.NewOracleClient(s.T())

				mockServer.On("Prices", mock.Anything, mock.Anything).Return(
					nil,
					nil,
				)

				return mockServer
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				return mockstrategies.NewCurrencyPairStrategy(s.T())
			},
			expectedResponse: &abcitypes.OracleVoteExtension{
				Prices: nil,
			},
		},
		{
			name: "oracle service returns an error",
			oracleService: func() client.OracleClient {
				mockServer := mocks.NewOracleClient(s.T())

				mockServer.On("Prices", mock.Anything, mock.Anything).Return(
					nil,
					fmt.Errorf("error"),
				)

				return mockServer
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				return mockstrategies.NewCurrencyPairStrategy(s.T())
			},
			expectedResponse: &abcitypes.OracleVoteExtension{
				Prices: nil,
			},
		},
		{
			name: "currency pair id strategy returns an error",
			oracleService: func() client.OracleClient {
				mockServer := mocks.NewOracleClient(s.T())

				mockServer.On("Prices", mock.Anything, mock.Anything).Return(
					&servicetypes.QueryPricesResponse{
						Prices: multiplePrices,
					},
					nil,
				)

				return mockServer
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				cps := mockstrategies.NewCurrencyPairStrategy(s.T())

				cps.On("ID", mock.Anything, btcUSD).Return(uint64(0), fmt.Errorf("error"))
				cps.On("ID", mock.Anything, ethUSD).Return(uint64(1), nil)
				cps.On("GetEncodedPrice", mock.Anything, ethUSD, twoHundred).Return(twoHundred.Bytes(), nil)

				return cps
			},
			expectedResponse: &abcitypes.OracleVoteExtension{
				Prices: map[uint64][]byte{
					1: twoHundred.Bytes(),
				},
			},
		},
		{
			name: "currency pair price strategy returns an error",
			oracleService: func() client.OracleClient {
				mockServer := mocks.NewOracleClient(s.T())

				mockServer.On("Prices", mock.Anything, mock.Anything).Return(
					&servicetypes.QueryPricesResponse{
						Prices: multiplePrices,
					},
					nil,
				)

				return mockServer
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				cps := mockstrategies.NewCurrencyPairStrategy(s.T())

				cps.On("ID", mock.Anything, btcUSD).Return(uint64(0), nil)
				cps.On("GetEncodedPrice", mock.Anything, btcUSD, oneHundred).Return(nil, fmt.Errorf("error"))

				cps.On("ID", mock.Anything, ethUSD).Return(uint64(1), nil)
				cps.On("GetEncodedPrice", mock.Anything, ethUSD, twoHundred).Return(twoHundred.Bytes(), nil)

				return cps
			},
			expectedResponse: &abcitypes.OracleVoteExtension{
				Prices: map[uint64][]byte{
					1: twoHundred.Bytes(),
				},
			},
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			codec := codec.NewCompressionVoteExtensionCodec(
				codec.NewDefaultVoteExtensionCodec(),
				codec.NewZLibCompressor(),
			)

			h := ve.NewVoteExtensionHandler(
				log.NewTestLogger(s.T()),
				tc.oracleService(),
				time.Second*1,
				tc.currencyPairStrategy(),
				codec,
				preblock.NoOpPreBlocker(),
				servicemetrics.NewNopMetrics(),
			)

			req := &cometabci.RequestExtendVote{}
			if tc.extendVoteRequest != nil {
				req = tc.extendVoteRequest()
			}
			resp, err := h.ExtendVoteHandler()(s.ctx, req)
			if !tc.expectedError {
				if resp == nil || len(resp.VoteExtension) == 0 {
					return
				}
				s.Require().NoError(err)
				s.Require().NotNil(resp)
				ve, err := codec.Decode(resp.VoteExtension)
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedResponse.Prices, ve.Prices)
			} else {
				s.Require().Error(err)
			}
		})
	}
}

func (s *VoteExtensionTestSuite) TestVerifyVoteExtension() {
	codec := codec.NewCompressionVoteExtensionCodec(
		codec.NewDefaultVoteExtensionCodec(),
		codec.NewZLibCompressor(),
	)

	cases := []struct {
		name                 string
		getReq               func() *cometabci.RequestVerifyVoteExtension
		currencyPairStrategy func() *mockstrategies.CurrencyPairStrategy
		expectedResponse     *cometabci.ResponseVerifyVoteExtension
		expectedError        bool
	}{
		{
			name: "nil request returns error",
			getReq: func() *cometabci.RequestVerifyVoteExtension {
				return nil
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				return mockstrategies.NewCurrencyPairStrategy(s.T())
			},
			expectedResponse: nil,
			expectedError:    true,
		},
		{
			name: "empty vote extension",
			getReq: func() *cometabci.RequestVerifyVoteExtension {
				return &cometabci.RequestVerifyVoteExtension{}
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				return mockstrategies.NewCurrencyPairStrategy(s.T())
			},
			expectedResponse: &cometabci.ResponseVerifyVoteExtension{
				Status: cometabci.ResponseVerifyVoteExtension_ACCEPT,
			},
			expectedError: false,
		},
		{
			name: "malformed bytes",
			getReq: func() *cometabci.RequestVerifyVoteExtension {
				return &cometabci.RequestVerifyVoteExtension{
					VoteExtension: []byte("malformed"),
				}
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				return mockstrategies.NewCurrencyPairStrategy(s.T())
			},
			expectedResponse: &cometabci.ResponseVerifyVoteExtension{
				Status: cometabci.ResponseVerifyVoteExtension_REJECT,
			},
			expectedError: true,
		},
		{
			name: "valid vote extension",
			getReq: func() *cometabci.RequestVerifyVoteExtension {
				prices := map[uint64][]byte{
					0: oneHundred.Bytes(),
					1: twoHundred.Bytes(),
				}

				ve, err := testutils.CreateVoteExtensionBytes(
					prices,
					codec,
				)
				s.Require().NoError(err)

				return &cometabci.RequestVerifyVoteExtension{
					VoteExtension: ve,
					Height:        1,
				}
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				cpStrategy := mockstrategies.NewCurrencyPairStrategy(s.T())

				cpStrategy.On("FromID", mock.Anything, uint64(0)).Return(btcUSD, nil)
				cpStrategy.On("GetDecodedPrice", mock.Anything, btcUSD, oneHundred.Bytes()).Return(oneHundred, nil)

				cpStrategy.On("FromID", mock.Anything, uint64(1)).Return(ethUSD, nil)
				cpStrategy.On("GetDecodedPrice", mock.Anything, ethUSD, twoHundred.Bytes()).Return(twoHundred, nil)

				return cpStrategy
			},
			expectedResponse: &cometabci.ResponseVerifyVoteExtension{
				Status: cometabci.ResponseVerifyVoteExtension_ACCEPT,
			},
			expectedError: false,
		},
		{
			name: "invalid vote extension with bad id",
			getReq: func() *cometabci.RequestVerifyVoteExtension {
				prices := map[uint64][]byte{
					0: oneHundred.Bytes(),
				}

				ve, err := testutils.CreateVoteExtensionBytes(
					prices,
					codec,
				)
				s.Require().NoError(err)

				return &cometabci.RequestVerifyVoteExtension{
					VoteExtension: ve,
					Height:        1,
				}
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				cpStrategy := mockstrategies.NewCurrencyPairStrategy(s.T())

				cpStrategy.On("FromID", mock.Anything, uint64(0)).Return(btcUSD, fmt.Errorf("error"))

				return cpStrategy
			},
			expectedResponse: &cometabci.ResponseVerifyVoteExtension{
				Status: cometabci.ResponseVerifyVoteExtension_REJECT,
			},
			expectedError: true,
		},
		{
			name: "invalid vote extension with bad price",
			getReq: func() *cometabci.RequestVerifyVoteExtension {
				prices := map[uint64][]byte{
					0: oneHundred.Bytes(),
				}

				ve, err := testutils.CreateVoteExtensionBytes(
					prices,
					codec,
				)
				s.Require().NoError(err)

				return &cometabci.RequestVerifyVoteExtension{
					VoteExtension: ve,
					Height:        1,
				}
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				cpStrategy := mockstrategies.NewCurrencyPairStrategy(s.T())

				cpStrategy.On("FromID", mock.Anything, uint64(0)).Return(btcUSD, nil)
				cpStrategy.On("GetDecodedPrice", mock.Anything, btcUSD, oneHundred.Bytes()).Return(nil, fmt.Errorf("error"))

				return cpStrategy
			},
			expectedResponse: &cometabci.ResponseVerifyVoteExtension{
				Status: cometabci.ResponseVerifyVoteExtension_REJECT,
			},
			expectedError: true,
		},
		{
			name: "vote extension with no prices",
			getReq: func() *cometabci.RequestVerifyVoteExtension {
				prices := map[uint64][]byte{}

				ve, err := testutils.CreateVoteExtensionBytes(
					prices,
					codec,
				)
				s.Require().NoError(err)

				return &cometabci.RequestVerifyVoteExtension{
					VoteExtension: ve,
					Height:        1,
				}
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				return mockstrategies.NewCurrencyPairStrategy(s.T())
			},
			expectedResponse: &cometabci.ResponseVerifyVoteExtension{
				Status: cometabci.ResponseVerifyVoteExtension_ACCEPT,
			},
			expectedError: false,
		},
		{
			name: "vote extension with malformed prices",
			getReq: func() *cometabci.RequestVerifyVoteExtension {
				prices := map[uint64][]byte{
					0: make([]byte, 34),
				}

				ve, err := testutils.CreateVoteExtensionBytes(
					prices,
					codec,
				)
				s.Require().NoError(err)

				return &cometabci.RequestVerifyVoteExtension{
					VoteExtension: ve,
					Height:        1,
				}
			},
			currencyPairStrategy: func() *mockstrategies.CurrencyPairStrategy {
				return mockstrategies.NewCurrencyPairStrategy(s.T())
			},
			expectedResponse: &cometabci.ResponseVerifyVoteExtension{
				Status: cometabci.ResponseVerifyVoteExtension_REJECT,
			},
			expectedError: true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			handler := ve.NewVoteExtensionHandler(
				log.NewTestLogger(s.T()),
				mocks.NewOracleClient(s.T()),
				time.Second*1,
				tc.currencyPairStrategy(),
				codec,
				preblock.NoOpPreBlocker(),
				servicemetrics.NewNopMetrics(),
			).VerifyVoteExtensionHandler()

			resp, err := handler(s.ctx, tc.getReq())
			s.Require().Equal(tc.expectedResponse, resp)

			if tc.expectedError {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *VoteExtensionTestSuite) TestExtendVoteLatency() {
	m := metricsmocks.NewMetrics(s.T())
	os := mocks.NewOracleClient(s.T())
	handler := ve.NewVoteExtensionHandler(
		log.NewTestLogger(s.T()),
		os,
		time.Second*1,
		mockstrategies.NewCurrencyPairStrategy(s.T()),
		codec.NewDefaultVoteExtensionCodec(),
		preblock.NoOpPreBlocker(),
		m,
	)

	// mock
	os.On("Prices", mock.Anything, mock.Anything).Return(
		&servicetypes.QueryPricesResponse{
			Prices:    map[string]string{},
			Timestamp: time.Now(),
		},
		nil,
	).Run(func(_ mock.Arguments) {
		// sleep to simulate latency
		time.Sleep(100 * time.Millisecond)
	})

	m.On("ObserveABCIMethodLatency", servicemetrics.ExtendVote, mock.Anything).Run(func(args mock.Arguments) {
		latency := args.Get(1).(time.Duration)
		s.Require().True(latency > 100*time.Millisecond)
	})
	m.On("AddABCIRequest", servicemetrics.ExtendVote, servicemetrics.Success{})
	_, err := handler.ExtendVoteHandler()(s.ctx, &cometabci.RequestExtendVote{
		Height: 1,
		Txs:    [][]byte{},
	})
	s.Require().NoError(err)
}

func (s *VoteExtensionTestSuite) TestExtendVoteStatus() {
	s.Run("test nil request", func() {
		mockMetrics := metricsmocks.NewMetrics(s.T())
		handler := ve.NewVoteExtensionHandler(
			log.NewTestLogger(s.T()),
			nil,
			time.Second*1,
			nil,
			nil,
			preblock.NoOpPreBlocker(),
			mockMetrics,
		)
		expErr := slinkyabci.NilRequestError{
			Handler: servicemetrics.ExtendVote,
		}
		mockMetrics.On("ObserveABCIMethodLatency", servicemetrics.ExtendVote, mock.Anything)
		mockMetrics.On("AddABCIRequest", servicemetrics.ExtendVote, expErr)

		handler.ExtendVoteHandler()(s.ctx, nil)
	})

	s.Run("test panic", func() {
		mockMetrics := metricsmocks.NewMetrics(s.T())
		handler := ve.NewVoteExtensionHandler(
			log.NewTestLogger(s.T()),
			nil,
			time.Second*1,
			nil,
			nil,
			func(_ sdk.Context, _ *cometabci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
				panic("panic")
			},
			mockMetrics,
		)
		expErr := ve.ErrPanic{
			Err: fmt.Errorf("panic"),
		}
		mockMetrics.On("ObserveABCIMethodLatency", servicemetrics.ExtendVote, mock.Anything)
		mockMetrics.On("AddABCIRequest", servicemetrics.ExtendVote, expErr)

		_, err := handler.ExtendVoteHandler()(s.ctx, &cometabci.RequestExtendVote{})
		s.Require().Error(err, expErr)
	})

	s.Run("test pre-blocker failure", func() {
		mockMetrics := metricsmocks.NewMetrics(s.T())
		preBlockError := fmt.Errorf("pre-blocker failure")
		handler := ve.NewVoteExtensionHandler(
			log.NewTestLogger(s.T()),
			nil,
			time.Second*1,
			nil,
			nil,
			func(_ sdk.Context, _ *cometabci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
				return nil, preBlockError
			},
			mockMetrics,
		)
		expErr := ve.PreBlockError{
			Err: preBlockError,
		}
		mockMetrics.On("ObserveABCIMethodLatency", servicemetrics.ExtendVote, mock.Anything)
		mockMetrics.On("AddABCIRequest", servicemetrics.ExtendVote, expErr)

		handler.ExtendVoteHandler()(s.ctx, &cometabci.RequestExtendVote{})
	})

	s.Run("test client failure", func() {
		mockMetrics := metricsmocks.NewMetrics(s.T())
		clientError := fmt.Errorf("client failure")
		mockClient := mocks.NewOracleClient(s.T())
		handler := ve.NewVoteExtensionHandler(
			log.NewTestLogger(s.T()),
			mockClient,
			time.Second*1,
			nil,
			nil,
			func(_ sdk.Context, _ *cometabci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
				return nil, nil
			},
			mockMetrics,
		)
		expErr := ve.OracleClientError{
			Err: clientError,
		}
		mockMetrics.On("ObserveABCIMethodLatency", servicemetrics.ExtendVote, mock.Anything)
		mockMetrics.On("AddABCIRequest", servicemetrics.ExtendVote, expErr)
		mockClient.On("Prices", mock.Anything, &servicetypes.QueryPricesRequest{}).Return(nil, clientError)

		handler.ExtendVoteHandler()(s.ctx, &cometabci.RequestExtendVote{})
	})

	s.Run("test price transformation failures", func() {
		mockMetrics := metricsmocks.NewMetrics(s.T())
		transformationError := fmt.Errorf("incorrectly formatted CurrencyPair: BTCETH")
		mockClient := mocks.NewOracleClient(s.T())
		handler := ve.NewVoteExtensionHandler(
			log.NewTestLogger(s.T()),
			mockClient,
			time.Second*1,
			nil,
			nil,
			func(_ sdk.Context, _ *cometabci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
				return nil, nil
			},
			mockMetrics,
		)
		expErr := ve.TransformPricesError{
			Err: transformationError,
		}
		mockMetrics.On("ObserveABCIMethodLatency", servicemetrics.ExtendVote, mock.Anything)
		mockMetrics.On("AddABCIRequest", servicemetrics.ExtendVote, expErr)
		mockClient.On("Prices", mock.Anything, &servicetypes.QueryPricesRequest{}).Return(&servicetypes.QueryPricesResponse{
			Prices: map[string]string{
				"BTCETH": "1000",
			},
		}, nil)

		handler.ExtendVoteHandler()(s.ctx, &cometabci.RequestExtendVote{})
	})

	s.Run("test codec failures", func() {
		mockMetrics := metricsmocks.NewMetrics(s.T())
		codecError := fmt.Errorf("codec error")
		mockClient := mocks.NewOracleClient(s.T())
		codec := codecmocks.NewVoteExtensionCodec(s.T())
		handler := ve.NewVoteExtensionHandler(
			log.NewTestLogger(s.T()),
			mockClient,
			time.Second*1,
			nil,
			codec,
			func(_ sdk.Context, _ *cometabci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
				return nil, nil
			},
			mockMetrics,
		)
		expErr := slinkyabci.CodecError{
			Err: codecError,
		}
		mockMetrics.On("ObserveABCIMethodLatency", servicemetrics.ExtendVote, mock.Anything)
		mockMetrics.On("AddABCIRequest", servicemetrics.ExtendVote, expErr)
		mockClient.On("Prices", mock.Anything, &servicetypes.QueryPricesRequest{}).Return(&servicetypes.QueryPricesResponse{
			Prices: map[string]string{},
		}, nil)
		codec.On("Encode", abcitypes.OracleVoteExtension{
			Prices: map[uint64][]byte{},
		}).Return(nil, codecError)

		handler.ExtendVoteHandler()(s.ctx, &cometabci.RequestExtendVote{})
	})

	s.Run("test success", func() {
		mockMetrics := metricsmocks.NewMetrics(s.T())
		mockClient := mocks.NewOracleClient(s.T())
		codec := codecmocks.NewVoteExtensionCodec(s.T())
		handler := ve.NewVoteExtensionHandler(
			log.NewTestLogger(s.T()),
			mockClient,
			time.Second*1,
			nil,
			codec,
			func(_ sdk.Context, _ *cometabci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
				return nil, nil
			},
			mockMetrics,
		)
		mockMetrics.On("ObserveABCIMethodLatency", servicemetrics.ExtendVote, mock.Anything)
		mockMetrics.On("AddABCIRequest", servicemetrics.ExtendVote, servicemetrics.Success{})
		mockClient.On("Prices", mock.Anything, &servicetypes.QueryPricesRequest{}).Return(&servicetypes.QueryPricesResponse{
			Prices: map[string]string{},
		}, nil)
		codec.On("Encode", abcitypes.OracleVoteExtension{
			Prices: map[uint64][]byte{},
		}).Return(nil, nil)

		_, err := handler.ExtendVoteHandler()(s.ctx, &cometabci.RequestExtendVote{})
		s.Require().NoError(err)
	})
}

func (s *VoteExtensionTestSuite) TestVerifyVoteExtensionStatus() {
	s.Run("nil request", func() {
		mockMetrics := metricsmocks.NewMetrics(s.T())
		handler := ve.NewVoteExtensionHandler(
			log.NewTestLogger(s.T()),
			nil,
			time.Second*1,
			nil,
			nil,
			preblock.NoOpPreBlocker(),
			mockMetrics,
		)
		expErr := slinkyabci.NilRequestError{
			Handler: servicemetrics.VerifyVoteExtension,
		}
		mockMetrics.On("ObserveABCIMethodLatency", servicemetrics.VerifyVoteExtension, mock.Anything)
		mockMetrics.On("AddABCIRequest", servicemetrics.VerifyVoteExtension, expErr)

		_, err := handler.VerifyVoteExtensionHandler()(s.ctx, nil)
		s.Require().Error(err, expErr)
	})

	s.Run("codec error", func() {
		mockMetrics := metricsmocks.NewMetrics(s.T())
		codecError := fmt.Errorf("codec error")
		codec := codecmocks.NewVoteExtensionCodec(s.T())
		handler := ve.NewVoteExtensionHandler(
			log.NewTestLogger(s.T()),
			nil,
			time.Second*1,
			nil,
			codec,
			preblock.NoOpPreBlocker(),
			mockMetrics,
		)
		expErr := slinkyabci.CodecError{
			Err: codecError,
		}
		mockMetrics.On("ObserveABCIMethodLatency", servicemetrics.VerifyVoteExtension, mock.Anything)
		mockMetrics.On("AddABCIRequest", servicemetrics.VerifyVoteExtension, expErr)
		codec.On("Decode", mock.Anything).Return(abcitypes.OracleVoteExtension{}, codecError)

		_, err := handler.VerifyVoteExtensionHandler()(s.ctx, &cometabci.RequestVerifyVoteExtension{
			VoteExtension: []byte{},
		})
		s.Require().Error(err, expErr)
	})

	s.Run("invalid vote-extension", func() {
		mockMetrics := metricsmocks.NewMetrics(s.T())

		codec := codecmocks.NewVoteExtensionCodec(s.T())
		length := 34
		transformErr := fmt.Errorf("price bytes are too long: %d", length)
		handler := ve.NewVoteExtensionHandler(
			log.NewTestLogger(s.T()),
			nil,
			time.Second*1,
			nil,
			codec,
			preblock.NoOpPreBlocker(),
			mockMetrics,
		)
		expErr := ve.ValidateVoteExtensionError{
			Err: transformErr,
		}
		mockMetrics.On("ObserveABCIMethodLatency", servicemetrics.VerifyVoteExtension, mock.Anything)
		mockMetrics.On("AddABCIRequest", servicemetrics.VerifyVoteExtension, expErr)
		codec.On("Decode", mock.Anything).Return(abcitypes.OracleVoteExtension{
			Prices: map[uint64][]byte{
				1: make([]byte, length),
			},
		}, nil)

		_, err := handler.VerifyVoteExtensionHandler()(s.ctx, &cometabci.RequestVerifyVoteExtension{
			VoteExtension: []byte{},
		})
		s.Require().Error(err, expErr)
	})

	s.Run("success", func() {
		mockMetrics := metricsmocks.NewMetrics(s.T())

		codec := codecmocks.NewVoteExtensionCodec(s.T())

		handler := ve.NewVoteExtensionHandler(
			log.NewTestLogger(s.T()),
			nil,
			time.Second*1,
			nil,
			codec,
			preblock.NoOpPreBlocker(),
			mockMetrics,
		)

		mockMetrics.On("ObserveABCIMethodLatency", servicemetrics.VerifyVoteExtension, mock.Anything)
		mockMetrics.On("AddABCIRequest", servicemetrics.VerifyVoteExtension, servicemetrics.Success{})
		mockMetrics.On("ObserveMessageSize", servicemetrics.VoteExtension, mock.Anything)

		codec.On("Decode", mock.Anything).Return(abcitypes.OracleVoteExtension{}, nil)

		_, err := handler.VerifyVoteExtensionHandler()(s.ctx, &cometabci.RequestVerifyVoteExtension{
			VoteExtension: []byte{},
		})
		s.Require().NoError(err)
	})
}

func (s *VoteExtensionTestSuite) TestVoteExtensionSize() {
	mockMetrics := metricsmocks.NewMetrics(s.T())

	mockClient := mocks.NewOracleClient(s.T())
	codec := codecmocks.NewVoteExtensionCodec(s.T())
	handler := ve.NewVoteExtensionHandler(
		log.NewTestLogger(s.T()),
		mockClient,
		time.Second*1,
		nil,
		codec,
		func(_ sdk.Context, _ *cometabci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
			return nil, nil
		},
		mockMetrics,
	)

	voteExtension := make([]byte, 100)

	// mock metrics calls
	mockMetrics.On("ObserveABCIMethodLatency", servicemetrics.VerifyVoteExtension, mock.Anything)
	mockMetrics.On("AddABCIRequest", servicemetrics.VerifyVoteExtension, servicemetrics.Success{})
	mockMetrics.On("ObserveMessageSize", servicemetrics.VoteExtension, 100)

	// mock codec calls
	codec.On("Decode", mock.Anything).Return(abcitypes.OracleVoteExtension{
		Prices: map[uint64][]byte{},
	}, nil)

	handler.VerifyVoteExtensionHandler()(s.ctx, &cometabci.RequestVerifyVoteExtension{
		VoteExtension: voteExtension,
	})
}
