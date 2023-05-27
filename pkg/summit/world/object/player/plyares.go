package player

type Players []*Player

func (pp *Players) Add(p *Player) {
	*pp = append(*pp, p)
}
