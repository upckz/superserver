package socket

import (
    "fmt"
    "github.com/orcaman/concurrent-map"
)

type ServerManger struct {
    serverMap cmap.ConcurrentMap
}

type GameManger struct {
    gameMap cmap.ConcurrentMap
}

type LevelManger struct {
    levelMap cmap.ConcurrentMap
}

type LogicManger struct {
    logicMap cmap.ConcurrentMap
}

func NewLogicManger() *LogicManger {
    l := &LogicManger{
        logicMap: cmap.New(),
    }
    return l
}

func (logic *LogicManger) AddServer(svid int, connid int) {
    key := fmt.Sprint("logic", svid)
    logic.logicMap.Set(key, connid)
}

func (logic *LogicManger) GetConnId(svid int) int {
    key := fmt.Sprint("logic", svid)
    connid, ok := logic.logicMap.Get(key)
    if ok {
        return connid.(int)
    }
    return -1
}

func (logic *LogicManger) Remove(svid int) {
    key := fmt.Sprint("logic", svid)
    logic.logicMap.Remove(key)
}

func NewLevelManger() *LevelManger {
    l := &LevelManger{
        levelMap: cmap.New(),
    }
    return l
}

func (ln *LevelManger) Add(level int, logic *LogicManger) {
    key := fmt.Sprint("level", level)
    if ln != nil {
        ln.levelMap.Set(key, logic)
    }
}

func (ln *LevelManger) FindServer(level int) *LogicManger {
    key := fmt.Sprint("level", level)
    logic, ok := ln.levelMap.Get(key)
    if ok {
        return logic.(*LogicManger)
    }
    return nil
}

func (ll *LevelManger) AddServer(level int, svid int, id int) {
    logic := ll.FindServer(level)
    if logic == nil {
        logic = NewLogicManger()
        ll.Add(level, logic)
    }
    logic.AddServer(svid, id)
}

func (ln *LevelManger) Remove(level int, svid int) {
    logic := ln.FindServer(level)
    if logic != nil {
        logic.Remove(svid)
    }
    if len(ln.levelMap) == 0 {
        key := fmt.Sprint("level", level)
        ln.levelMap.Remove(key)
    }
}

func (ln *LevelManger) GetConnId(level int, svid int) int {
    logic := ln.FindServer(level)
    if logic != nil {
        return logic.GetConnId(svid)
    }
    return -1
}

func NewGameManger() *GameManger {
    g := &GameManger{
        gameMap: cmap.New(),
    }
    return g
}

func (g *GameManger) FindServer(gameid int) *LevelManger {
    key := fmt.Sprint("game", gameid)
    l, ok := g.gameMap.Get(key)
    if ok {
        return l.(*LevelManger)
    }
    return nil
}

func (g *GameManger) Add(gameid int, ln *LevelManger) {
    key := fmt.Sprint("game", gameid)
    if ln != nil {
        g.gameMap.Set(key, ln)
    }
}

func (g *GameManger) Remove(gameid int, level int, svid int) {
    ln := g.FindServer(gameid)
    if ln != nil {
        ln.Remove(level, svid)
    }
    if len(g.gameMap) == 0 {
        key := fmt.Sprint("game", gameid)
        g.gameMap.Remove(key)
    }
}

func (g *GameManger) GetConnId(gameid int, level int, svid int) int {
    ln := g.FindServer(gameid)
    if ln != nil {
        return ln.GetConnId(level, svid)
    }
    return -1
}

func (g *GameManger) AddServer(gameid int, level int, svid int, connid int) {

    ln := g.FindServer(gameid)
    if ln == nil {
        ln = NewLevelManger()
        g.Add(gameid, ln)
    }
    ln.AddServer(level, svid, connid)
}

func NewServerManger() *ServerManger {
    s := &ServerManger{
        serverMap: cmap.New(),
    }
    return s
}

func (s *ServerManger) InsertServer(svrType int, gameid int, level int, svid int, connid int) {

    game := s.FindServer(svrType)
    if game == nil {
        game = NewGameManger()
        s.Add(svrType, game)
    }
    game.AddServer(gameid, level, svid, connid)

}

func (s *ServerManger) Add(svrType int, game *GameManger) {
    key := fmt.Sprint("svr", svrType)
    if game != nil {
        s.serverMap.Set(key, game)
    }
}

func (s *ServerManger) FindServer(svrType int) *GameManger {
    key := fmt.Sprint("svr", svrType)
    game, ok := s.serverMap.Get(key)
    if ok {
        return game.(*GameManger)
    }
    return nil
}

func (s *ServerManger) Remove(svrType int, gameid int, level int, svid int) {
    ln := s.FindServer(svrType)
    if ln != nil {
        ln.Remove(gameid, level, svid)
    }
    if len(s.serverMap) == 0 {
        key := fmt.Sprint("svr", svrType)
        s.serverMap.Remove(key)
    }
}

func (s *ServerManger) GetConnId(svrType int, gameid int, level int, svid int) int {
    ln := s.FindServer(svrType)
    if ln != nil {
        return ln.GetConnId(gameid, level, svid)
    }
    return -1
}
