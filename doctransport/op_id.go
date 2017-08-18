package doctransport

import "strings"

func makeOperationID(httpMethod, path string) string {
	httpMethod = strings.ToLower(httpMethod)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	switch httpMethod {
	case "patch", "put":
		httpMethod = "update"
	case "post":
		httpMethod = "create"
	}

	idParts := []string{httpMethod}
	pathParts := strings.Split(path, "/")

	for len(pathParts) > 0 {
		forceSingular := false
		skip := 1
		if len(pathParts) >= 2 && strings.HasPrefix(pathParts[1], "{") && strings.HasSuffix(pathParts[1], "}") {
			forceSingular = true
			skip = 2
		}
		if len(pathParts) == 1 && httpMethod == "create" {
			forceSingular = true
		}

		part := strings.Title(pathParts[0])
		part = strings.Replace(part, "-", "", -1)

		if forceSingular {
			part = strings.TrimSuffix(part, "s")
		}

		idParts = append(idParts, part)
		pathParts = pathParts[skip:]
	}

	return strings.Join(idParts, "")
}
