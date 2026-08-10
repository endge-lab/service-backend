package entities

// FolderEntityType возвращает физический тип папок для коллекции документов.
func FolderEntityType(collection string) string {
	if collection == "streams" {
		return "queries"
	}
	return collection
}

// RootFolderIdentity возвращает единый корень папок для коллекции документов.
func RootFolderIdentity(collection string) string {
	return "root-" + FolderEntityType(collection)
}
