/*
   Go Language Raspberry Pi Interface
   (c) Copyright David Thorpe 2016-2018
   All Rights Reserved
   Documentation http://djthorpe.github.io/gopi/
   For Licensing and Usage information, please see LICENSE.md
*/
package main

import (
	"context"
	"fmt"
	"io"
	"strconv"

	// Frameworks
	"github.com/djthorpe/gopi"
	"github.com/djthorpe/sensors"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type CommandFunc struct {
	Fn          func(*MiHomeApp, []string) error
	Description string
}

type MiHomeApp struct {
	log    gopi.Logger
	gpio   gopi.GPIO
	mihome sensors.MiHome
	ctx    context.Context
	cancel context.CancelFunc
	args   []string
}

////////////////////////////////////////////////////////////////////////////////
// GLOBAL VARIABLES

var (
	COMMANDS = map[string]CommandFunc{
		"reset_gpio":   CommandFunc{ResetGPIO, "Reset GPIO"},
		"reset_radio":  CommandFunc{ResetRadio, "Reset RFM69 Radio"},
		"measure_temp": CommandFunc{MeasureTemp, "Measure Temperature"},
		"on":           CommandFunc{TransmitOn, "On TX (optionally use 1,2,3,4 as additional argument)"},
		"off":          CommandFunc{TransmitOff, "Off TX (optionally use 1,2,3,4 as additional argument)"},
		"receive_ook":  CommandFunc{ReceiveOOK, "Receive data in OOK mode"},
		"receive_fsk":  CommandFunc{ReceiveFSK, "Receive data in FSK mode"},
	}
)

////////////////////////////////////////////////////////////////////////////////
// METHODS

func CommandNames() []string {
	commands := make([]string, 0, len(COMMANDS))
	for k, _ := range COMMANDS {
		commands = append(commands, k)
	}
	return commands
}

func PrintCommands(out io.Writer) {
	fmt.Fprintf(out, "Commands:\n")
	for k, v := range COMMANDS {
		fmt.Fprintf(out, "  %-10s\t%s\n", k, v.Description)
	}
}

////////////////////////////////////////////////////////////////////////////////
// APP Commands

func NewApp(app *gopi.AppInstance) *MiHomeApp {
	this := new(MiHomeApp)
	this.log = app.Logger
	this.gpio = app.ModuleInstance("gpio").(gopi.GPIO)
	this.mihome = app.ModuleInstance("sensors/ener314rt").(sensors.MiHome)

	if this.gpio == nil || this.mihome == nil {
		return nil
	}
	if args := app.AppFlags.Args(); len(args) == 0 {
		return nil
	} else {
		this.args = args
	}

	return this
}

func (this *MiHomeApp) NewContext() context.Context {
	this.Cancel()
	this.ctx, this.cancel = context.WithCancel(context.Background())
	return this.ctx
}

func (this *MiHomeApp) Cancel() {
	if this.cancel != nil {
		this.cancel()
		this.cancel = nil
	}
}

func (this *MiHomeApp) Run() error {
	cmd := this.args[0]
	other := this.args[1:]
	this.log.Info(cmd)
	if command, exists := COMMANDS[cmd]; exists == false {
		return gopi.ErrHelp
	} else if err := command.Fn(this, other); err != nil {
		return err
	}
	return nil
}

func ResetGPIO(this *MiHomeApp, args []string) error {
	// Ensure pins are in correct state for SPI0
	this.gpio.SetPinMode(gopi.GPIOPin(7), gopi.GPIO_OUTPUT)
	this.gpio.SetPinMode(gopi.GPIOPin(8), gopi.GPIO_OUTPUT)
	this.gpio.SetPinMode(gopi.GPIOPin(9), gopi.GPIO_ALT0)
	this.gpio.SetPinMode(gopi.GPIOPin(10), gopi.GPIO_ALT0)
	this.gpio.SetPinMode(gopi.GPIOPin(11), gopi.GPIO_ALT0)
	// Success
	return nil
}

func ResetRadio(this *MiHomeApp, args []string) error {
	if len(args) > 0 {
		return gopi.ErrHelp
	}
	return this.mihome.ResetRadio()
}

func MeasureTemp(this *MiHomeApp, args []string) error {
	if len(args) > 0 {
		return gopi.ErrHelp
	}
	if celcius, err := this.mihome.MeasureTemperature(); err != nil {
		return err
	} else {
		fmt.Printf("Temperature=%v\u00B0C\n", celcius)
	}

	// Return success
	return nil
}

func TransmitOn(this *MiHomeApp, args []string) error {
	if sockets, err := toSockets(args); err != nil {
		return err
	} else if err := this.mihome.On(sockets...); err != nil {
		return err
	}
	// Return success
	return nil
}

func TransmitOff(this *MiHomeApp, args []string) error {
	if sockets, err := toSockets(args); err != nil {
		return err
	} else if err := this.mihome.Off(sockets...); err != nil {
		return err
	}
	// Return success
	return nil
}

func ReceiveOOK(this *MiHomeApp, args []string) error {
	if len(args) > 0 {
		return gopi.ErrHelp
	} else if err := this.mihome.Receive(this.NewContext(), sensors.MIHOME_MODE_CONTROL); err != nil {
		return err
	}

	// Return success
	return nil
}

func ReceiveFSK(this *MiHomeApp, args []string) error {
	if len(args) > 0 {
		return gopi.ErrHelp
	} else if err := this.mihome.Receive(this.NewContext(), sensors.MIHOME_MODE_MONITOR); err != nil {
		return err
	}

	// Return success
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// UTILTY FUNCTIONS

func toSockets(args []string) ([]uint, error) {
	// Where there are no arguments, assume 'all'
	if len(args) == 0 {
		return nil, nil
	}

	// Else read uintegers into an array
	ret := make([]uint, 0, len(args))
	for _, value := range args {
		if uint_value, err := strconv.ParseUint(value, 10, 64); err != nil {
			return nil, err
		} else {
			ret = append(ret, uint(uint_value))
		}
	}

	// Return success
	return ret, nil
}