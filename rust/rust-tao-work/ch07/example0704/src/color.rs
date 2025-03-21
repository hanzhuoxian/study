use std::fmt::Display;

pub struct ColoredString {
    pub input: String,
    pub fg_color: String,
    pub bg_color: String,
}

pub trait Colorize {
    const FG_RED: &'static str = "31";
    const BG_YELLOW: &'static str = "34";
    fn red(self) -> ColoredString;
    fn on_yellow(self) -> ColoredString;
}

impl Default for ColoredString {
    fn default() -> Self {
        ColoredString {
            input: String::default(),
            fg_color: String::default(),
            bg_color: String::default(),
        }
    }
}
impl<'a> Colorize for ColoredString {
    fn red(self) -> ColoredString {
        ColoredString {
            fg_color: String::from(ColoredString::FG_RED),
            ..self
        }
    }
    fn on_yellow(self) -> ColoredString {
        ColoredString {
            bg_color: String::from(ColoredString::BG_YELLOW),
            ..self
        }
    }
}

impl<'a> Colorize for &'a str {
    fn red(self) -> ColoredString {
        ColoredString {
            input: String::from(self),
            fg_color: String::from(ColoredString::FG_RED),
            ..ColoredString::default()
        }
    }

    fn on_yellow(self) -> ColoredString {
        ColoredString {
            input: String::from(self),
            bg_color: String::from(ColoredString::BG_YELLOW),
            ..ColoredString::default()
        }
    }
}

impl ColoredString {
    fn compute_style(&self) -> String {
        let mut res = String::from("\x1B[");
        let mut has_wrote = false;
        if !self.bg_color.is_empty() {
            res.push_str(&self.bg_color);
            has_wrote = true;
        }

        if !self.fg_color.is_empty() {
            if has_wrote {
                res.push_str(";");
            }
            res.push_str(&self.fg_color);
        }
        res.push('m');
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
