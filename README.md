# ![Kelvin](https://user-images.githubusercontent.com/512174/37403613-b56e883a-278f-11e8-848c-5366515e920d.png)
[![GitHub release](https://img.shields.io/github/release/stefanwichmann/kelvin.svg)](https://github.com/stefanwichmann/kelvin/releases)
[![Github All Releases](https://img.shields.io/github/downloads/stefanwichmann/kelvin/total.svg)](https://github.com/stefanwichmann/kelvin/releases)
[![Build Status](https://travis-ci.org/stefanwichmann/kelvin.svg?branch=master)](https://travis-ci.org/stefanwichmann/kelvin)
[![Go Report Card](https://goreportcard.com/badge/github.com/stefanwichmann/kelvin)](https://goreportcard.com/report/github.com/stefanwichmann/kelvin)
[![license](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/stefanwichmann/kelvin/blob/master/LICENSE)
[![donate](https://img.shields.io/badge/donate-PayPal-yellow.svg)](https://paypal.me/stefanwichmann)

# Meet Kelvin
Kelvin is a little helper bot who will automate the lights in your house. Its job is to adjust the color temperature and brightness in your home based on your local sunrise and sunset times and custom intervals defined by you. Think of it as [f.lux](https://justgetflux.com/) or Apple's Night Shift for your home.

Imagine your lights shine in an energetic but not too bright blue color to get you started in the early morning. On sunrise your lights will change to a more natural color temperature to reflect the sunlight outside. On sunset they will slowly fade to a warmer and softer color scheme perfectly suited to Netflix and chill. When it's time to go to bed Kelvin will reduce the intensity even more to get you into a sleepy mood. It will keep this reduced setting through the night so you don't get blinded by bright lights if you have to get up at night...

# Features
- Adjust the color temperature and brightness of your lights based on the local sunrise and sunset times
- Define fine grained daily schedules to fit your personal needs throughout the day for every single room
- Define a default startup color and brightness for your lights
- Gradual light transitions you won't even notice
- Works with smart switches as well as conventional switches
- Activate via Hue Scene or automatically for every light you turn on
- Respects manual light changes until a light is switched off and on again
- Auto upgrade to seamlessly deliver improvements to you
- Small, self contained binary with sane defaults and no dependencies to get you started right away
- Free and open source

# Getting started
If you want to give Kelvin a try, there are some things you will need to benefit from its services:
- [ ] Supported **Philips Hue** (or compatible) lights
- [ ] A configured **Philips Hue** bridge
- [ ] A permanently running computer connected to your network (See [Raspberry Pi](#raspberry-pi))

Got all these? Great, let's get started!

# Installation
1. Download the latest version of Kelvin from the [Releases](https://github.com/stefanwichmann/kelvin/releases) page.
2. Extract the Kelvin archive.
3. Start Kelvin by double-clicking `kelvin.exe` on windows or by typing `./kelvin` in your terminal on macOS, Linux and other Unix-based systems.
   You should see an output similar to the following snippet:
   ```
   2017/03/22 10:45:41 Kelvin v1.1.0 starting up... 🚀
   2017/03/22 10:45:41 Looking for updates...
   2017/03/22 10:45:41 ⚙ Default configuration generated
   2017/03/22 10:45:41 ⌘ No bridge configuration found. Starting local discovery...
   2017/03/22 10:45:44 ⌘ Found bridge. Starting user registration.
   PLEASE PUSH THE BLUE BUTTON ON YOUR HUE BRIDGE...
   ```
4. Now you have to allow Kelvin to talk to your bridge by pushing the blue button on top of your physical Hue bridge. Kelvin will wait one minute for you to push the button. If you didn't make it in time just start it again with step 3.
5. Once you pushed the button you should see something like:
   ```
   2017/03/22 10:45:41 Kelvin v1.1.0 starting up... 🚀
   2017/03/22 10:45:41 Looking for updates...
   2017/03/22 10:45:41 ⚙ Default configuration generated
   2017/03/22 10:45:41 ⌘ No bridge configuration found. Starting local discovery...
   2017/03/22 10:45:44 ⌘ Found bridge. Starting user registration.
   PLEASE PUSH THE BLUE BUTTON ON YOUR HUE BRIDGE... Success!
   2017/03/22 10:45:59 💡 Devices found on current bridge:
   2017/03/22 10:45:59 | Name                 |  ID | On    | Dimmable | Temperature | Color |
   2017/03/22 10:45:59 | Dining table         |   5 | false | true     | true        | true  |
   2017/03/22 10:45:59 | Power outlet         |   6 | false | false    | false       | false |
   2017/03/22 10:45:59 | Window               |   1 | false | true     | true        | true  |
   2017/03/22 10:45:59 | Kitchen              |   2 | false | true     | true        | true  |
   2017/03/22 10:45:59 | Couch                |   3 | false | true     | true        | true  |
   2017/03/22 10:45:59 | Desk                 |   4 | false | true     | false       | true  |
   2017/03/22 10:45:59 Device Power outlet doesn't support any functionality we use. Exclude it from unnecessary polling.
   2017/03/22 10:45:59 🌍 Location not configured. Detecting by IP
   2017/03/22 10:45:59 🌍 Detected location: Hamburg, Germany (53.5553, 9.995).
   2017/03/22 10:45:59 💡 Starting cyclic update for Desk
   2017/03/22 10:45:59 💡 Starting cyclic update for Window
   2017/03/22 10:45:59 💡 Starting cyclic update for Dining table
   2017/03/22 10:45:59 💡 Starting cyclic update for Kitchen
   2017/03/22 10:45:59 💡 Starting cyclic update for Couch
   2017/03/22 10:45:59 Configuring intervals for Wednesday March 22 2017
   2017/03/22 10:45:59 Managing lights for interval 06:21 to 18:40
   ```
6. Wohoo! Kelvin is up and running! Well done!
7. Kelvin is now managing your lights and will gradually adjust the color temperature and brightness for you. Give it a try by switching lights on and off to see how Kelvin reacts. If you want to adjust the default schedule to your needs, just read on and edit the configuration.

# Docker
As an alternative to manual installation you can also pull the official [docker image](https://hub.docker.com/r/stefanwichmann/kelvin/) from docker hub.

- Get the image by running ```docker pull stefanwichmann/kelvin```
- Start a container via ```docker run -d -e TZ=Europe/Berlin -p 8080:8080 stefanwichmann/kelvin``` (replace Europe/Berlin with your local [timezone](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones))
- ```docker ps``` should now report your running container
- Run ```docker logs {CONTAINER_ID}``` to see the kelvin output (You can get the valid ID from ```docker ps```)
- To adjust the configuration you should use the web interface running at ```http://{DOCKER_HOST_IP}:8080/```.
- If you want to keep your configuration over the lifetime of your container, you can map the folder ```/etc/opt/kelvin/``` to your host filesystem. If you alter the configuration you have to restart Kelvin through the web interface or by running ```docker restart {CONTAINER_ID}```.

# Configuration
Kelvin will create it's configuration file `config.json` in the current directory and store all necessary information to operate in it. By default it is fully usable and looks like this:

```
{
  "bridge": {
    "ip": "192.168.10.37",
    "username": "lbCDGagZZ7JEYQX5iGxrjMIx2jIROgpXfsSjHmCv"
  },
  "location": {
    "latitude": 53.5553,
    "longitude": 9.995
  },
  "schedules": [
    {
      "name": "default",
      "associatedDeviceIDs": [1,2,3,4,5,6],
      "enableWhenLightsAppear": true,
      "defaultColorTemperature": 2750,
      "defaultBrightness": 100,
      "beforeSunrise": [
        {
          "time": "4:00",
          "colorTemperature": 2000,
          "brightness": 60
        }
      ],
      "afterSunset": [
        {
          "time": "20:00",
          "colorTemperature": 2300,
          "brightness": 80
        },
        {
          "time": "22:00",
          "colorTemperature": 2000,
          "brightness": 60
        }
      ]
    }
  ]
}
```
As the configuration file is a simple text file in JSON format you can display and edit it with you favorite text editor. Just make sure you keep the JSON structure valid. If something goes wrong fix it using [JSONLint](http://jsonlint.com/) or just delete the `config.json` and let Kelvin generate a configuration from scratch.

The configuration contains the following fields:

| Name | Description |
| ---- | ----------- |
| bridge | This element contains the IP and username of your Philips Hue bridge. Both values are usually obtained automatically. If the lookup fails you can fill in this details by hand. [Learn more](https://github.com/stefanwichmann/kelvin/wiki/Manual-bridge-configuration)|
| location | This element contains the latitude and longitude of your location on earth. Both values are determined by your public IP. If this fails, is inaccurate or you want to change it manually just fill in your own coordinates. |
| schedules | This element contains an array of all your configured schedules. See below for a detailed description of a schedule configuration. |

Each schedule must be configured in the following format:

| Name | Description |
| ---- | ----------- |
| name | The name of this schedule. This is only used for better readability. |
| associatedDeviceIDs | A list of all devices/lights that should be managed according to this schedule. Kelvin will print an overview of all your devices on startup. You should use this to associate your lights with the right schedule. *ATTENTION: Every light should be associated to only one schedule. If you skip an ID this device will be ignored.* |
| enableWhenLightsAppear | If this element is set to `true` Kelvin will be activated automatically whenever you switch an associated light on. If set to `false` Kelvin won't take over until you enable a [Kelvin Scene](#kelvin-scenes) or activate it via web interface. |
| defaultColorTemperature | This default color temperature will be used between sunrise and sunset. Valid values are between 2000K and 6500K. See [Wikipedia](https://en.wikipedia.org/wiki/Color_temperature) for reference values. If you set this value to -1 Kelvin will ignore the color temperature and you can change it manually.|
| defaultBrightness | This default brightness value will be used between sunrise and sunset. Valid values are between 0% and 100%. If you set this value to -1 Kelvin will ignore the brightness and you can change it manually.|
| beforeSunrise | This element contains a list of timestamps and their configuration you want to set between midnight and sunrise of any given day. The *time* value must follow the `hh:mm` format. *colorTemperature* and *brightness* must follow the same rules as the default values. |
| afterSunset | This element contains a list of timestamps and their configuration you want to set between sunset and midnight of any given day. The *time* value must follow the `hh:mm` format. *colorTemperature* and *brightness* must follow the same rules as the default values. |

After altering the configuration you have to restart Kelvin. Just kill the running instance (`Ctrl+C` or `kill $PID`) or send a HUP signal (`kill -s HUP $PID`) to the process to restart (unix only).

# Kelvin Scenes
Kelvin has the ability to detect certain light scenes you have programmed in your hue system. If you activate one of these Kelvin scenes it will take control of the light and manage it for you. You can use this feature to reactivate Kelvin after manually changing the light state or to associate Kelvin with a certain button on your Hue Tap for example.

In order to use this feature you have to create a scene in your Hue System via a Hue app. The name of this new scene has to contain the word `kelvin` and the name of the schedule you want to control. Once you saved this scene Kelvin will associate all relevant lights to it and update the state every minute to fit your schedule. Now you can simple activate this scene whenever you want to active Kelvin.

Let's look at an example:
- Let's assume you have a schedule called `livingroom` which should be activated only on the second tap of your Hue Tap.
- Start a Hue app on your smartphone and create a new scene called `Activate Kelvin in Livingroom` or `Livingroom (Kelvin)`. The exact name doesn't matter as long as the words `kelvin` and the name of the schedule are part of this scene name.
- Associate the new scene to the second tap on your Hue Tap and set the configuration value `enableWhenLightsAppear` to `false` in the schedule `livingroom`.
- Restart Kelvin to activate the new configuration.
- From now on Kelvin will only take control of the lights in the schedule `livingroom` if you activate the scene on the second tap.

# Raspberry Pi
A [Raspberry Pi](https://www.raspberrypi.org/) is the **perfect** device to run Kelvin on. It's cheap, it's small and it consumes very little energy. Recently the [Raspberry Pi Zero W](https://www.raspberrypi.org/products/pi-zero-w/) was released which makes your Kelvin hardware look like this (plus a power cord):

![Raspberry Pi Zero W](https://www.raspberrypi.org/wp-content/uploads/2017/02/zero-wireless.png)

But any other model of the Raspberry Pi will be sufficient. To set up Kelvin on a Raspberry Pi follow the installation guide [here](https://www.raspberrypi.org/documentation/installation/). Once your Pi is up and running (booting, connected to your network and the internet) just download the latest `linux-arm` release and follow the steps in [Installation](#installation).

# Systemd setup on a RaspberryPi
Running Kelvin as a systemd process provides an easily managed background process.

There are a couple assumptions made:
* [Raspberry Pi](https://www.raspberrypi.org/) is the hardware
* Running [Rasbian OS](https://www.raspberrypi.org/downloads/raspbian/) installed with defaults
* Install path for Kelvin binary is `/home/pi/kelvin/kelvin`
* Logged in as `pi` user, and have `sudo` user permissions

## Setup
```shell
# Fetch release
wget https://github.com/stefanwichmann/kelvin/releases/download/v1.1.9/kelvin-linux-arm-v1.1.9.tar.gz -O /tmp/kelvin-arm.tar.gz

# Create user to run as
sudo adduser --system --group --shell /bin/nologin --no-create-home --home /opt/kelvin kelvin

# Install
sudo mkdir -p /opt/kelvin
cd /opt/kelvin
sudo tar -xvzf /tmp/kelvin-arm.tar.gz
sudo mv kelvin-linux-arm*/* .
sudo rmdir kelvin-linux-arm*
sudo chown -R kelvin:kelvin /opt/kelvin

# Create service file for systemd
sudo cp etc/kelvin.service /etc/systemd/system/kelvin.service

# Start, then press hue button. Restart if necessary
sudo systemctl start kelvin

# Start on boot
sudo systemctl enable kelvin

# Confirm status
sudo systemctl status kelvin

# Clean up
rm /tmp/kelvin-arm.tar.gz

# Edit config to taste
sudo -u kelvin -e /opt/kelvin/config.json
sudo systemctl restart kelvin

# Read Logs
journalctl -fu kelvin.service
```

If you are using Kelvin on a different system with Systemd you have to adjust the `kelvin.service` file according to your needs.

# Troubleshooting
If anything goes wrong keep calm and follow these steps:

1. Make sure the Philips Hue bridge is configured and working in your network. Kelvin will need it to communicate with your lights. If you got the Hue app running on your smartphone you should be fine. Otherwise follow the Philips Hue manual to configure your lights.

2. To identify the IP address of your bridge open [this](https://www.meethue.com/api/nupnp) link in your browser. After you got the IP address enter `http://<bridge IP address>/debug/clip.html` into your browser. You should see the debug page of you hue bridge. If this fails please follow the Philips Hue manual to configure your bridge.

3. Make sure the Philips Hue bridge is reachable from the computer Kelvin will run on. Enter the command `ping <bridge IP address>` in a terminal window or on a remote console. You should see packages reaching the destination IP address. If this fails you might have a network issue.

4. Make sure you downloaded the latest release for your operating system and CPU architecture. If you are not sure stick to the most appropriate `amd64` release or `arm` if you are using a Raspberry Pi.

5. If all this doesn't help, feel free to open an [issue](https://github.com/stefanwichmann/kelvin/issues) on github.

# How Kelvin works
In order to decide if Kelvin suits your needs and works in your setup, it helps to understand it's inner workings and behavior. In a nutshells Kelvin uses your Philips Hue bridge to talk to all the Hue lights in your home and will automatically configure them according to the schedules in your configuration file. In order to do this it will request the current state of every light every two seconds. For this state Kelvin differentiates three possible scenarios:

1. ***The light is turned on:*** Kelvin will calculate the appropriate color temperature and brightness, send it to the light and safe this state.
2. ***The light is turned on but it's state was changed since the last update:*** Kelvin detects that you have manually changed the state (for example by activating a custom scene) and will stop managing the state for you.
3. ***The light is turned off:*** Kelvin will clear the last known state and do nothing.

# Development & Participation
If you want to tinker with Kelvin and it's inner workings, feel free to do so. To get started you can simple clone the main repository into your `GOPATH` by executing the following commands:
```
cd $GOPATH
go get -v github.com/stefanwichmann/kelvin
```
Make sure you have set up your [go](https://www.golang.org) development environment by following the steps in the official [documentation](https://golang.org/doc/).

If you have ideas how to improve Kelvin I will gladly accept pull requests from your forks or discuss them with you through an [issue](https://github.com/stefanwichmann/kelvin/issues).
