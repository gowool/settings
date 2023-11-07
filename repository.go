package settings

import (
	"context"
	"fmt"
)

type NamespaceRepository interface {
	FindByName(ctx context.Context, name string) (*Namespace, error)
	DeleteByName(ctx context.Context, name string) error
	Save(ctx context.Context, namespace *Namespace) error
}

type PreferenceRepository interface {
	FindByNsAndKey(ctx context.Context, namespace, key string) (*Preference, error)
	DeleteByNsAndKey(ctx context.Context, namespace, key string) error
	Save(ctx context.Context, preference *Preference) error
}

type (
	Cache interface {
		Set(ctx context.Context, key string, value interface{}, tags ...string) error
		Get(ctx context.Context, key string, value interface{}) error
		DelByKey(ctx context.Context, key string) error
		DelByTag(ctx context.Context, tag string) error
	}

	CacheNamespaceRepository struct {
		NamespaceRepository
		Cache Cache
	}

	CachePreferenceRepository struct {
		PreferenceRepository
		Cache Cache
	}
)

func (r CacheNamespaceRepository) FindByName(ctx context.Context, name string) (namespace *Namespace, err error) {
	key := "settings:ns:name:" + name

	if err = r.Cache.Get(ctx, key, &namespace); err == nil {
		return
	}

	if namespace, err = r.NamespaceRepository.FindByName(ctx, name); err != nil {
		return
	}

	_ = r.Cache.Set(ctx, key, namespace, "settings:ns:tag:"+name)

	return
}

func (r CacheNamespaceRepository) DeleteByName(ctx context.Context, name string) error {
	defer r.del(ctx, name)

	return r.NamespaceRepository.DeleteByName(ctx, name)
}

func (r CacheNamespaceRepository) Save(ctx context.Context, namespace *Namespace) error {
	defer func() { r.del(ctx, namespace.Name) }()

	return r.NamespaceRepository.Save(ctx, namespace)
}

func (r CacheNamespaceRepository) del(ctx context.Context, name string) {
	_ = r.Cache.DelByTag(ctx, "settings:ns:tag:"+name)
}

func (r CachePreferenceRepository) FindByNsAndKey(ctx context.Context, namespace, key string) (preference *Preference, err error) {
	k := fmt.Sprintf("settings:pref:ns:key:%s:%s", namespace, key)

	if err = r.Cache.Get(ctx, k, &preference); err == nil {
		return
	}

	if preference, err = r.PreferenceRepository.FindByNsAndKey(ctx, namespace, key); err != nil {
		return
	}

	_ = r.Cache.Set(ctx, k, preference, fmt.Sprintf("settings:pref:tag:%s:%s", namespace, key))

	return
}

func (r CachePreferenceRepository) DeleteByNsAndKey(ctx context.Context, namespace, key string) error {
	defer r.del(ctx, namespace, key)

	return r.PreferenceRepository.DeleteByNsAndKey(ctx, namespace, key)
}

func (r CachePreferenceRepository) Save(ctx context.Context, preference *Preference) error {
	defer func() { r.del(ctx, preference.Namespace, preference.Key) }()

	return r.PreferenceRepository.Save(ctx, preference)
}

func (r CachePreferenceRepository) del(ctx context.Context, namespace, key string) {
	_ = r.Cache.DelByTag(ctx, fmt.Sprintf("settings:pref:tag:%s:%s", namespace, key))
}
