package shoppinglist

type ShoppingItem struct {
	Name     string
	ItemType ShoppingItemType
}

type ShoppingItemType string

const (
	Fruit      ShoppingItemType = "fruit"
	Veg        ShoppingItemType = "vegetable"
	Dairy      ShoppingItemType = "dairy"
	Meat       ShoppingItemType = "meat"
	MeatFree   ShoppingItemType = "meat_free"
	Seafood    ShoppingItemType = "seafood"
	Bakery     ShoppingItemType = "bakery"
	Pantry     ShoppingItemType = "pantry"
	Snacks     ShoppingItemType = "snacks"
	Frozen     ShoppingItemType = "frozen"
	Drinks     ShoppingItemType = "drinks"
	Cleaning   ShoppingItemType = "cleaning"
	Toiletries ShoppingItemType = "toiletries"
	Baby       ShoppingItemType = "baby"
	Health     ShoppingItemType = "health"
	Household  ShoppingItemType = "household"
	Spices     ShoppingItemType = "spices"
	Condiments ShoppingItemType = "condiments"
)
