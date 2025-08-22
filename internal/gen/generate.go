package gen

// Run go generate to automatically generate provider, data source and resource types
// from the intermediate representation JSON file `ir.json`.
//go:generate tfplugingen-framework generate provider --package gen --output .
//go:generate tfplugingen-framework generate data-sources --package gen --output .
//go:generate tfplugingen-framework generate resources --package gen --output .
