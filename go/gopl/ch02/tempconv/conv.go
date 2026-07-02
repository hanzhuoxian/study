package tempconv

// 摄氏温度转华氏温度
func CToF(c Celsius) Fahrenheit { return Fahrenheit(c*9/5 + 32) }

// 华氏温度转摄氏温度
func FToC(f Fahrenheit) Celsius { return Celsius((f - 32) * 5 / 9) }

func CToK(c Celsius) Kelvin { return Kelvin(c + 273.15) }
func KToC(k Kelvin) Celsius { return Celsius(k - 273.15) }
