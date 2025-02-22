package types_test

import (
	"math/big"
	"testing"

	"cosmossdk.io/math"
	cmtabci "github.com/cometbft/cometbft/abci/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"

	"github.com/skip-mev/slinky/x/alerts/types"
	oracletypes "github.com/skip-mev/slinky/x/oracle/types"
)

func TestMsgAlertValidateBasic(t *testing.T) {
	type testCase struct {
		name string
		types.MsgAlert
		valid bool
	}

	testCases := []testCase{
		{
			"if the alert is invalid, the message is invalid",
			*types.NewMsgAlert(types.Alert{
				Height:       0,
				Signer:       "",
				CurrencyPair: oracletypes.NewCurrencyPair("BTC", "USD"),
			}),
			false,
		},
		{
			"if the alert is valid, the message is valid",
			*types.NewMsgAlert(types.NewAlert(0, sdk.AccAddress("cosmos1"), oracletypes.NewCurrencyPair("BTC", "USD"))),
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.MsgAlert.ValidateBasic()
			if tc.valid && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if !tc.valid && err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestMsgAlertGetSigners(t *testing.T) {
	signer := sdk.AccAddress("cosmos1")

	// create a message with signer
	msg := types.NewMsgAlert(types.NewAlert(0, signer, oracletypes.NewCurrencyPair("BTC", "USD")))
	signers := msg.GetSigners()
	assert.Equal(t, []sdk.AccAddress{signer}, signers)
}

func TestMsgConclusion(t *testing.T) {
	invalidConclusionAny, err := codectypes.NewAnyWithValue(&types.MultiSigConclusion{})
	assert.NoError(t, err)

	validConclusionAny, err := codectypes.NewAnyWithValue(&types.MultiSigConclusion{
		ExtendedCommitInfo: cmtabci.ExtendedCommitInfo{},
		Alert: types.Alert{
			Height:       1,
			Signer:       sdk.AccAddress("cosmos1").String(),
			CurrencyPair: oracletypes.NewCurrencyPair("BTC", "USD"),
		},
		PriceBound: types.PriceBound{
			High: big.NewInt(1).String(),
			Low:  big.NewInt(0).String(),
		},
		Signatures: []types.Signature{
			{
				sdk.AccAddress("cosmos1").String(),
				nil,
			},
		},
		Status: false,
	})
	assert.NoError(t, err)

	t.Run("test validate basic", func(t *testing.T) {
		cases := []struct {
			name  string
			msg   types.MsgConclusion
			valid bool
		}{
			{
				"invalid signer address",
				types.MsgConclusion{
					Signer:     "invalid",
					Conclusion: validConclusionAny,
				},
				false,
			},
			{
				"nil conclusion",
				types.MsgConclusion{
					Signer:     sdk.AccAddress("cosmos1").String(),
					Conclusion: nil,
				},
				false,
			},
			{
				"invalid conclusion",
				types.MsgConclusion{
					Signer:     sdk.AccAddress("cosmos1").String(),
					Conclusion: invalidConclusionAny,
				},
				false,
			},
			{
				"valid conclusion",
				types.MsgConclusion{
					Signer:     sdk.AccAddress("cosmos1").String(),
					Conclusion: validConclusionAny,
				},
				true,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.msg.ValidateBasic()
				if tc.valid && err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !tc.valid && err == nil {
					t.Errorf("expected error, got nil")
				}
			})
		}
	})

	t.Run("test get signers", func(t *testing.T) {
		signer := sdk.AccAddress("cosmos1")
		msg := types.MsgConclusion{
			Signer:     signer.String(),
			Conclusion: validConclusionAny,
		}

		signers := msg.GetSigners()
		assert.Equal(t, []sdk.AccAddress{signer}, signers)
	})
}

func TestMsgUpdateParams(t *testing.T) {
	cases := []struct {
		name  string
		msg   types.MsgUpdateParams
		valid bool
	}{
		{
			"invalid signer address",
			types.MsgUpdateParams{
				Authority: "false",
				Params:    types.DefaultParams("denom", nil),
			},
			false,
		},
		{
			"invalid params",
			types.MsgUpdateParams{
				Authority: sdk.AccAddress("cosmos1").String(),
				Params: types.Params{
					AlertParams: types.AlertParams{
						Enabled:    false,
						BondAmount: sdk.NewCoin("denom", math.NewInt(1)),
					},
				},
			},
			false,
		},
		{
			"valid message",
			types.MsgUpdateParams{
				Authority: sdk.AccAddress("cosmos1").String(),
				Params:    types.DefaultParams("denom", nil),
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.valid && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if !tc.valid && err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}
