package main

// For posterity.
//
// func TestCreateGroup(t *testing.T) {
// 	t.Error("HI")
// 	b := &db.Binder{}
// 	query := `
// WITH cg AS (
//   INSERT INTO cgroup(id) VALUES (DEFAULT) RETURNING *
// ), i AS (
//   INSERT INTO user_cgroup(uid, cgid)
//     SELECT ` + b.Bind(1) + `, id FROM cg
// )
// SELECT id FROM cg`
// 	var g Group
// 	err := db.DB.Get(&g, query, b.Items...)
// 	log.Println(g, err)
// }
