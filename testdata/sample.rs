use std::collections::HashMap;
use std::fmt;

pub struct Registry {
    items: HashMap<String, Item>,
    name: String,
}

pub trait Processor {
    fn process(&self, input: &[u8]) -> Result<Vec<u8>, Error>;
    fn name(&self) -> &str;
}

impl Registry {
    pub fn new(name: String) -> Self {
        Self {
            items: HashMap::new(),
            name,
        }
    }

    pub fn register(&mut self, name: String, item: Item) {
        self.items.insert(name, item);
    }

    pub fn get(&self, name: &str) -> Option<&Item> {
        self.items.get(name)
    }
}

impl fmt::Display for Registry {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "Registry({}, {} items)", self.name, self.items.len())
    }
}

pub fn init_logging(level: &str) {
    env_logger::Builder::new()
        .filter_level(level.parse().unwrap())
        .init();
}

pub enum Status {
    Active,
    Inactive,
    Pending(String),
}
