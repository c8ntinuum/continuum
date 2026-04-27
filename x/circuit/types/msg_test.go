package types

import (
	"testing"

	"github.com/stretchr/testify/suite"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type MsgsTestSuite struct {
	suite.Suite
}

func TestMsgsTestSuite(t *testing.T) {
	suite.Run(t, new(MsgsTestSuite))
}

func (suite *MsgsTestSuite) TestMsgUpdateParamsValidateBasic() {
	validParams := DefaultParams()

	testCases := []struct {
		name    string
		msg     *MsgUpdateParams
		expPass bool
	}{
		{
			name: "fail - invalid authority address",
			msg: &MsgUpdateParams{
				Authority: "invalid",
				Params:    &validParams,
			},
			expPass: false,
		},
		{
			name: "fail - nil params",
			msg: &MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    nil,
			},
			expPass: false,
		},
		{
			name: "pass - valid msg",
			msg: &MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    &validParams,
			},
			expPass: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := tc.msg.ValidateBasic()
			if tc.expPass {
				suite.NoError(err)
				return
			}
			suite.Error(err)
		})
	}
}
