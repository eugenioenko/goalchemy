package rt

func LibCryptoPublicPEM(t *Task, key *Key) {
	r0, r1 := key.PublicPEM()
	t.RV = []any{r0, r1}
}
