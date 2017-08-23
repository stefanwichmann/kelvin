function kelvinToRGB (temp, out) {
  if (!Array.isArray(out)) {
    out = [0, 0, 0]
  }

  temp = temp / 100
  var red, blue, green

  if (temp <= 66) {
    red = 255
  } else {
    red = temp - 60
    red = 329.698727466 * Math.pow(red, -0.1332047592)
    if (red < 0) {
      red = 0
    }
    if (red > 255) {
      red = 255
    }
  }

  if (temp <= 66) {
    green = temp
    green = 99.4708025861 * Math.log(green) - 161.1195681661
    if (green < 0) {
      green = 0
    }
    if (green > 255) {
      green = 255
    }
  } else {
    green = temp - 60
    green = 288.1221695283 * Math.pow(green, -0.0755148492)
    if (green < 0) {
      green = 0
    }
    if (green > 255) {
      green = 255
    }
  }

  if (temp >= 66) {
    blue = 255
  } else {
    if (temp <= 19) {
      blue = 0
    } else {
      blue = temp - 10
      blue = 138.5177312231 * Math.log(blue) - 305.0447927307
      if (blue < 0) {
        blue = 0
      }
      if (blue > 255) {
        blue = 255
      }
    }
  }

  out[0] = Math.floor(red)
  out[1] = Math.floor(green)
  out[2] = Math.floor(blue)
  return out
}
