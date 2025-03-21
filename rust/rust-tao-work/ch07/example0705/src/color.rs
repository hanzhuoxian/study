use std::{fmt::Display, str::FromStr};

pub enum Color {
    Red,
    Yellow,
    Blue,
}

pub struct ColoredString {
    input: String,
    fg_color: Option<Color>,
    bg_color: Option<Color>,
}

impl Default for ColoredString {
    fn default() -> Self {
        ColoredString {
            input: String::default(),
            fg_color: None,
            bg_color: None,
        }
    }
}

impl Color {
    fn to_fg_str(&self) -> &str {
        match *self {
            Color::Red => "31",
            Color::Yellow => "33",
            Color::Blue => "34",
        }
    }

    fn to_bg_str(&self) -> &str {
        match *self {
            Color::Red => "41",
            Color::Yellow => "43",
            Color::Blue => "44",
        }
    }
}

impl FromStr for Color {
    type Err = ();
    fn from_str(src: &str) -> Result<Self, Self::Err> {
        let src = src.to_lowercase();
        match src.as_ref() {
            "red" => Ok(Color::Red),
            "yellow" => Ok(Color::Yellow),
            "blue" => Ok(Color::Blue),
            _ => Err(()),
        }
    }
}

impl<'a> From<&'a str> for Color {
    fn from(src: &str) -> Self {
        src.parse().unwrap_or(Color::Red)
    }
}

impl From<String> for Color {
    fn from(src: String) -> Self {
        src.parse().unwrap_or(Color::Red)
    }
}

pub trait Colorize {
    fn red(self) -> ColoredString;
    fn yellow(self) -> ColoredString;
    fn blue(self) -> ColoredString;
    fn color<S: Into<Color>>(self, color: S) -> ColoredString;
    fn on_red(self) -> ColoredString;
    fn on_yellow(self) -> ColoredString;
    fn on_blue(self) -> ColoredString;
    fn on_color<S: Into<Color>>(self, color: S) -> ColoredString;
}

impl Colorize for ColoredString {
    fn red(self) -> ColoredString {
        self.color(Color::Red)
    }
    fn yellow(self) -> ColoredString {
        self.color(Color::Yellow)
    }
    fn blue(self) -> ColoredString {
        self.color(Color::Blue)
    }
    fn color<S: Into<Color>>(mut self, color: S) -> ColoredString {
        self.fg_color = Some(color.into());
        self
    }
    fn on_red(self) -> ColoredString {
        self.on_color(Color::Red)
    }
    fn on_yellow(self) -> ColoredString {
        self.on_color(Color::Yellow)
    }
    fn on_blue(self) -> ColoredString {
        self.on_color(Color::Blue)
    }
    fn on_color<S: Into<Color>>(mut self, color: S) -> ColoredString {
        self.bg_color = Some(color.into());
        self
    }
}

impl Colorize for &str {
    fn red(self) -> ColoredString {
        self.color(Color::Red)
    }
    fn yellow(self) -> ColoredString {
        self.color(Color::Yellow)
    }
    fn blue(self) -> ColoredString {
        self.color(Color::Blue)
    }
    fn color<S: Into<Color>>(self, color: S) -> ColoredString {
        ColoredString {
            input: self.to_string(),
            fg_color: Some(color.into()),
            ..ColoredString::default()
        }
    }
    fn on_red(self) -> ColoredString {
        self.on_color(Color::Red)
    }
    fn on_yellow(self) -> ColoredString {
        self.on_color(Color::Yellow)
    }
    fn on_blue(self) -> ColoredString {
        self.on_color(Color::Blue)
    }
    fn on_color<S: Into<Color>>(self, color: S) -> ColoredString {
        ColoredString {
            input: self.to_string(),
            bg_color: Some(color.into()),
            ..ColoredString::default()
        }
    }
}

impl ColoredString {
    pub fn compute_style(&self) -> String {
        let mut res = String::from("\x1B[");
        let mut has_wrote = false;
        if let Some(bg_color) = &self.bg_color {
            if has_wrote {
                res.push_str(";");
            }
            res += bg_color.to_bg_str();
            has_wrote = true;
        }
        if let Some(fg_color) = &self.fg_color {
            if has_wrote {
                res.push_str(";");
            }
            res += fg_color.to_fg_str();
        }
        res.push_str("m");
        res
    }
}

impl Display for ColoredString {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let input = &self.input.clone();
        f.write_str(&self.compute_style())?;
        f.write_str(input)?;
        f.write_str("\x1B[0m")?;
        Ok(())
    }
}