package gojail

//JailGetByName queries the OS for Jail with name jailName
func JailGetByName(jailName string) (Jail, error) {
	jid, err := JailGetID(jailName)
	if err != nil {
		return nil, err
	}
	return &jail{
		jailID:   jid,
		jailName: jailName,
	}, nil
}

//JailGetByID queries the OS for Jail with jid jailID
func JailGetByID(jailID JailID) (Jail, error) {
	name, err := JailGetName(jailID)
	if err != nil {
		return nil, err
	}
	return &jail{
		jailID:   jailID,
		jailName: name,
	}, nil
}
