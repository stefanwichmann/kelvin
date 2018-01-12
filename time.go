// MIT License
//
// Copyright (c) 2018 Stefan Wichmann
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package main

import "time"
import "github.com/bt51/ntpclient"

const timeServer = "0.pool.ntp.org"
const maxTimeDifferenceInSeconds = 60

// IsLocalTimeValid validates the clock on the local machine via NTP.
// If the network time differs more than one minute from the local
// time it is considered invalid. It is up to the caller to continue
// anyway.
func IsLocalTimeValid() (bool, error) {
	networkTime, err := ntpclient.GetNetworkTime(timeServer, 123)
	if err != nil {
		return false, err
	}
	timeDifference := time.Until(*networkTime).Seconds()
	if timeDifference < 0 {
		timeDifference = timeDifference * -1
	}

	if timeDifference < maxTimeDifferenceInSeconds {
		return true, nil
	}
	return false, nil
}
