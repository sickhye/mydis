// Copyright 2017 Ross Peoples
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mydis

import (
	"sort"
	"strings"

	"github.com/deejross/mydis/pb"
	"github.com/deejross/mydis/util"
	"github.com/gogo/protobuf/proto"
	"golang.org/x/net/context"
)

// GetHash gets a hash from the cache.
func (s *Server) GetHash(ctx context.Context, key *pb.Key) (*pb.Hash, error) {
	res, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	h := &pb.Hash{}
	if err := proto.Unmarshal(res.Value, h); err != nil && strings.HasPrefix(err.Error(), "proto: can't skip unknown wire type") {
		return nil, util.ErrTypeMismatch
	} else if err != nil {
		return nil, err
	}
	return h, nil
}

// GetHashField gets the value of a hash field.
func (s *Server) GetHashField(ctx context.Context, hf *pb.HashField) (*pb.ByteValue, error) {
	h, err := s.GetHash(ctx, &pb.Key{Key: hf.Key})
	if err != nil {
		return nil, err
	}
	if b, ok := h.Value[hf.Field]; ok {
		return &pb.ByteValue{Value: b}, nil
	}
	return nil, util.ErrHashFieldNotFound
}

// GetHashFields gets a list of values from a hash.
func (s *Server) GetHashFields(ctx context.Context, hs *pb.HashFieldSet) (*pb.Hash, error) {
	h, err := s.GetHash(ctx, &pb.Key{Key: hs.Key})
	if err != nil {
		return nil, err
	}

	nh := &pb.Hash{Value: map[string][]byte{}}
	for _, field := range hs.Field {
		if b, ok := h.Value[field]; ok {
			nh.Value[field] = b
		}
	}
	return nh, nil
}

// HashHas determines if a hash has the given field.
func (s *Server) HashHas(ctx context.Context, hf *pb.HashField) (*pb.Bool, error) {
	h, err := s.GetHash(ctx, &pb.Key{Key: hf.Key})
	if err != nil {
		return nil, err
	}

	if _, ok := h.Value[hf.Field]; ok {
		return &pb.Bool{Value: true}, nil
	}
	return &pb.Bool{Value: false}, nil
}

// HashLength gets the number of fields in a hash.
func (s *Server) HashLength(ctx context.Context, key *pb.Key) (*pb.IntValue, error) {
	h, err := s.GetHash(ctx, key)
	if err != nil {
		return nil, err
	}
	return &pb.IntValue{Value: int64(len(h.Value))}, nil
}

// HashFields gets all fields in a hash.
func (s *Server) HashFields(ctx context.Context, key *pb.Key) (*pb.KeysList, error) {
	h, err := s.GetHash(ctx, key)
	if err != nil {
		return nil, err
	}

	lst := &pb.KeysList{Keys: []string{}}
	for field := range h.Value {
		lst.Keys = append(lst.Keys, field)
	}
	sort.Strings(lst.Keys)
	return lst, nil
}

// HashValues gets all values in a hash.
func (s *Server) HashValues(ctx context.Context, key *pb.Key) (*pb.List, error) {
	h, err := s.GetHash(ctx, key)
	if err != nil {
		return nil, err
	}

	keys := []string{}
	for key := range h.Value {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lst := &pb.List{Value: [][]byte{}}
	for _, key := range keys {
		b := h.Value[key]
		lst.Value = append(lst.Value, b)
	}
	return lst, nil
}

// SetHash sets a hash in the cache.
func (s *Server) SetHash(ctx context.Context, h *pb.Hash) (*pb.Null, error) {
	key := h.Key
	h.Key = ""
	b, err := proto.Marshal(h)
	if err != nil {
		return null, err
	}
	_, err = s.Set(ctx, &pb.ByteValue{Key: key, Value: b})
	return null, err
}

// SetHashField sets a single field in a hash, creates new hash if does not exist.
func (s *Server) SetHashField(ctx context.Context, hf *pb.HashField) (*pb.Null, error) {
	key := &pb.Key{Key: hf.Key}
	if _, err := s.Lock(ctx, key); err != nil {
		return null, err
	}

	h, err := s.GetHash(ctx, key)
	if err == util.ErrKeyNotFound {
		h = &pb.Hash{Value: map[string][]byte{}}
	} else if err != nil {
		s.Unlock(ctx, key)
		return null, err
	}
	h.Value[hf.Field] = hf.Value
	h.Key = hf.Key
	return s.UnlockThenSetHash(ctx, h)
}

// SetHashFields sets multiple fields in a hash, creates new hash if does not exist.
func (s *Server) SetHashFields(ctx context.Context, ah *pb.Hash) (*pb.Null, error) {
	key := &pb.Key{Key: ah.Key}
	if _, err := s.Lock(ctx, key); err != nil {
		return null, err
	}

	h, err := s.GetHash(ctx, key)
	if err == util.ErrKeyNotFound {
		h = &pb.Hash{Value: map[string][]byte{}}
	} else if err != nil {
		s.Unlock(ctx, key)
		return null, err
	}

	for key, b := range ah.Value {
		h.Value[key] = b
	}

	h.Key = ah.Key
	return s.UnlockThenSetHash(ctx, h)
}

// DelHashField removes a field from a hash.
func (s *Server) DelHashField(ctx context.Context, hf *pb.HashField) (*pb.Null, error) {
	key := &pb.Key{Key: hf.Key}
	if _, err := s.Lock(ctx, key); err != nil {
		return null, err
	}

	h, err := s.GetHash(ctx, key)
	if err != nil {
		s.Unlock(ctx, key)
		return null, err
	}
	delete(h.Value, hf.Field)
	h.Key = hf.Key
	return s.UnlockThenSetHash(ctx, h)
}
