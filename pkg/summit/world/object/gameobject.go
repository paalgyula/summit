package object

type GameObject struct {
	*Object
}

func NewGameObject() *GameObject {
	gobject := &GameObject{
		Object: NewObject(),
	}

	// m_objectType |= TYPEMASK_GAMEOBJECT
	// m_objectTypeId = TYPEID_GAMEOBJECT
	// 2.3.2 - 0x58
	// m_updateFlag = (UPDATEFLAG_LOWGUID | UPDATEFLAG_HIGHGUID | UPDATEFLAG_HAS_POSITION)

	// m_valuesCount = GAMEOBJECT_END

	return gobject
}
