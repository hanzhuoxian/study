use rltk::{GameState, RGB, Rltk, VirtualKeyCode};
use specs::prelude::*;
use specs_derive::Component;
use std::cmp::{max, min};

#[derive(Component)]
struct Position {
    x: i32,
    y: i32,
}

#[derive(Component)]
struct RenderAble {
    glyph: rltk::FontCharType,
    fg: RGB,
    bg: RGB,
}

struct state {
    ecs: World,
}

fn main() {
    let mut gs = state { ecs: World::new() };
    gs.ecs.register::<Position>();
    gs.ecs.register::<RenderAble>();

    gs.ecs
        .create_entity()
        .with(Position { x: 0, y: 0 })
        .with(RenderAble {
            glyph: rltk::to_cp437('@'),
            fg: RGB::named(rltk::YELLOW),
            bg: RGB::named(rltk::BLACK),
        })
        .build();

    for i in 0..10 {
        gs.ecs
            .create_entity()
            .with(Position { x: i * 7, y: 20 })
            .with(RenderAble {
                glyph: rltk::to_cp437('☺'),
                fg: RGB::named(rltk::RED),
                bg: RGB::named(rltk::BLACK),
            })
            .build();
    }
    println!("Hello, world!");
}
