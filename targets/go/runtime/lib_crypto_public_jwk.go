package rt

func LibCryptoPublicJWK(t *Task, key *Key) {
	r0, r1 := key.PublicJWK()
	t.RV = []any{r0, r1}
}
