# pets

Pet endpoints.

### GET `/pets`

Retrieve all pets.

**Responses**

- `200` — OK
  - `application/json`: [PetList](../schemas/PetList.md)

### POST `/pets`

Create a pet.

**Request body**

Required.

- `application/json`: [Pet](../schemas/Pet.md)

**Responses**

- `201` — Created
  - `application/json`: [Pet](../schemas/Pet.md)

### GET `/pets/{petId}`

Get a pet by id.

**Parameters**

- `petId` (path, required, string (uuid)) — Pet UUID.

**Responses**

- `200` — OK
  - `application/json`: [Pet](../schemas/Pet.md)
- `404` — Not Found
  - `application/json`: [Error](../schemas/Error.md)

