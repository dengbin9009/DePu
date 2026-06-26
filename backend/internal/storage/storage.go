package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/dengbin9009/DePu/backend/internal/game"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Save(g *game.Game) error {
	if g == nil {
		return errors.New("nil game")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	body, err := json.Marshal(g)
	if err != nil {
		return err
	}
	initialSnapshot := g.InitialSnapshotJSON
	if initialSnapshot == "" {
		if err = tx.QueryRow(`select snapshot from snapshots where game_id = ? and seq = 0`, g.ID).Scan(&initialSnapshot); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	if initialSnapshot == "" {
		initialSnapshot = string(body)
	}

	_, err = tx.Exec(`
		insert into games(id, ruleset_id, stage, version, snapshot, updated_at)
		values(?, ?, ?, ?, ?, ?)
		on conflict(id) do update set
			ruleset_id=excluded.ruleset_id,
			stage=excluded.stage,
			version=excluded.version,
			snapshot=excluded.snapshot,
			updated_at=excluded.updated_at
	`, g.ID, g.RuleSetID, string(g.Stage), g.Version, string(body), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	_, err = tx.Exec(`delete from actions where game_id = ?`, g.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`delete from snapshots where game_id = ?`, g.ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		insert into snapshots(game_id, seq, snapshot, created_at)
		values(?, ?, ?, ?)
	`, g.ID, 0, initialSnapshot, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	for _, action := range g.Actions {
		payload, marshalErr := json.Marshal(action.Payload)
		if marshalErr != nil {
			return marshalErr
		}
		summary, marshalErr := json.Marshal(action.StateSummary)
		if marshalErr != nil {
			return marshalErr
		}
		_, err = tx.Exec(`
			insert into actions(game_id, seq, stage, seat_no, type, amount, payload, summary, created_at)
			values(?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, g.ID, action.Seq, string(action.Stage), nullableSeat(action.SeatNo), string(action.Type), action.Amount, string(payload), string(summary), action.CreatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
	}
	for _, action := range g.Actions {
		snapshotJSON := action.SnapshotJSON
		if snapshotJSON == "" {
			snapshotJSON = string(body)
		}
		_, err = tx.Exec(`
			insert into snapshots(game_id, seq, snapshot, created_at)
			values(?, ?, ?, ?)
		`, g.ID, action.Seq, snapshotJSON, time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) Load(id string) (*game.Game, error) {
	var snapshot string
	if err := s.db.QueryRow(`select snapshot from games where id = ?`, id).Scan(&snapshot); err != nil {
		return nil, err
	}
	var g game.Game
	if err := json.Unmarshal([]byte(snapshot), &g); err != nil {
		return nil, err
	}
	_ = s.db.QueryRow(`select snapshot from snapshots where game_id = ? and seq = 0`, id).Scan(&g.InitialSnapshotJSON)
	return &g, nil
}

func (s *Store) History(id string) ([]game.Action, error) {
	rows, err := s.db.Query(`
		select seq, stage, coalesce(seat_no, 0), type, amount, payload, summary, created_at
		from actions
		where game_id = ?
		order by seq asc
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []game.Action
	for rows.Next() {
		var action game.Action
		var stage, typ, payload, summary, created string
		if err := rows.Scan(&action.Seq, &stage, &action.SeatNo, &typ, &action.Amount, &payload, &summary, &created); err != nil {
			return nil, err
		}
		action.Stage = game.Stage(stage)
		action.Type = game.ActionType(typ)
		_ = json.Unmarshal([]byte(payload), &action.Payload)
		_ = json.Unmarshal([]byte(summary), &action.StateSummary)
		action.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		actions = append(actions, action)
	}
	return actions, rows.Err()
}

func (s *Store) SnapshotAt(id string, seq int) (*game.Game, error) {
	var snapshot string
	if seq < 0 {
		return nil, errors.New("replay sequence out of range")
	}
	if seq == 0 {
		if err := s.db.QueryRow(`
			select snapshot from snapshots
			where game_id = ?
			order by seq asc
			limit 1
		`, id).Scan(&snapshot); err != nil {
			return nil, err
		}
	} else {
		var latest int
		if err := s.db.QueryRow(`select coalesce(max(seq), 0) from actions where game_id = ?`, id).Scan(&latest); err != nil {
			return nil, err
		}
		if seq > latest {
			return nil, errors.New("replay sequence out of range")
		}
		err := s.db.QueryRow(`
			select snapshot from snapshots
			where game_id = ? and seq = ?
		`, id, seq).Scan(&snapshot)
		if err != nil {
			return nil, err
		}
	}
	var g game.Game
	if err := json.Unmarshal([]byte(snapshot), &g); err != nil {
		return nil, err
	}
	_ = s.db.QueryRow(`select snapshot from snapshots where game_id = ? and seq = 0`, id).Scan(&g.InitialSnapshotJSON)
	return &g, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		create table if not exists games (
			id text primary key,
			ruleset_id text not null,
			stage text not null,
			version integer not null,
			snapshot text not null,
			updated_at text not null
		);
		create table if not exists actions (
			game_id text not null,
			seq integer not null,
			stage text not null,
			seat_no integer null,
			type text not null,
			amount integer not null,
			payload text not null,
			summary text not null,
			created_at text not null,
			primary key(game_id, seq)
		);
		create table if not exists snapshots (
			game_id text not null,
			seq integer not null,
			snapshot text not null,
			created_at text not null,
			primary key(game_id, seq)
		);
	`)
	return err
}

func nullableSeat(seatNo int) any {
	if seatNo == 0 {
		return nil
	}
	return seatNo
}
