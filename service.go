package settings

import (
	"context"
	"fmt"
)

var _ Service = &Loader{}

const (
	namespaceSystem  = "system"
	keyConfiguration = "configuration"
)

type Service interface {
	GetNamespaceValue(ctx context.Context, namespace string) (string, error)
	SetNamespaceValue(ctx context.Context, namespace, value string) error
	RemoveNamespaceValue(ctx context.Context, namespace string) error

	LoadConfigByNamespace(ctx context.Context, namespace string, config any) error
	SaveConfigByNamespace(ctx context.Context, namespace string, config any) error
	RemoveConfigByNamespace(ctx context.Context, namespace string) error

	LoadConfig(ctx context.Context, namespace string, config any) error
	SaveConfig(ctx context.Context, namespace string, config any) error
	RemoveConfig(ctx context.Context, namespace string) error
}

func NewService(
	nsRepo NamespaceRepository,
	prefRepo PreferenceRepository,
	configurer Configurer,
	cache Cache,
) Service {
	return ConfigurerLoader{
		Service: Loader{
			NsRepo: CacheNamespaceRepository{
				NamespaceRepository: nsRepo,
				Cache:               cache,
			},
			PrefRepo: CachePreferenceRepository{
				PreferenceRepository: prefRepo,
				Cache:                cache,
			},
		},
		Configurer: configurer,
		Cache:      cache,
	}
}

type Loader struct {
	NsRepo   NamespaceRepository
	PrefRepo PreferenceRepository
}

func (s Loader) GetNamespaceValue(ctx context.Context, namespace string) (string, error) {
	return s.getSystemValue(ctx, namespace)
}

func (s Loader) SetNamespaceValue(ctx context.Context, namespace, value string) error {
	return s.saveConfig(ctx, namespaceSystem, namespace, value)
}

func (s Loader) RemoveNamespaceValue(ctx context.Context, namespace string) error {
	return s.PrefRepo.DeleteByNsAndKey(ctx, namespaceSystem, namespace)
}

func (s Loader) LoadConfigByNamespace(ctx context.Context, namespace string, config any) error {
	value, err := s.GetNamespaceValue(ctx, namespace)
	if err != nil {
		return err
	}

	return s.LoadConfig(ctx, value, config)
}

func (s Loader) SaveConfigByNamespace(ctx context.Context, namespace string, config any) error {
	value, err := s.GetNamespaceValue(ctx, namespace)
	if err != nil {
		return err
	}

	return s.SaveConfig(ctx, value, config)
}

func (s Loader) RemoveConfigByNamespace(ctx context.Context, namespace string) error {
	value, err := s.GetNamespaceValue(ctx, namespace)
	if err != nil {
		return err
	}

	return s.RemoveConfig(ctx, value)
}

func (s Loader) LoadConfig(ctx context.Context, namespace string, config any) error {
	if _, err := s.NsRepo.FindByName(ctx, namespace); err != nil {
		return fmt.Errorf("namespace `%s` not found: %w", namespace, err)
	}

	pref, err := s.getConfiguration(ctx, namespace)
	if err != nil {
		return err
	}

	return pref.LoadValue(config)
}

func (s Loader) SaveConfig(ctx context.Context, namespace string, config any) error {
	return s.saveConfig(ctx, namespace, keyConfiguration, config)
}

func (s Loader) RemoveConfig(ctx context.Context, namespace string) error {
	if err := s.PrefRepo.DeleteByNsAndKey(ctx, namespace, keyConfiguration); err != nil {
		return err
	}

	return s.NsRepo.DeleteByName(ctx, namespace)
}

func (s Loader) getSystemValue(ctx context.Context, key string) (string, error) {
	pref, err := s.getSystemValueByKey(ctx, key)
	if err != nil {
		return "", err
	}

	var namespace string
	if err = pref.LoadValue(&namespace); err != nil {
		return "", fmt.Errorf("namespace of system preference `%s` not found: %w", key, err)
	}

	return namespace, nil
}

func (s Loader) getSystemValueByKey(ctx context.Context, key string) (pref *Preference, err error) {
	if pref, err = s.PrefRepo.FindByNsAndKey(ctx, namespaceSystem, key); err != nil {
		err = fmt.Errorf("system preference `%s` not found: %w", key, err)
	}
	return
}

func (s Loader) getConfiguration(ctx context.Context, namespace string) (pref *Preference, err error) {
	if pref, err = s.PrefRepo.FindByNsAndKey(ctx, namespace, keyConfiguration); err != nil {
		err = fmt.Errorf("configuration preference `%s` not found: %w", namespace, err)
	}
	return
}

func (s Loader) saveConfig(ctx context.Context, namespace, key string, config any) error {
	if err := s.NsRepo.Save(ctx, &Namespace{Name: namespace}); err != nil {
		return err
	}

	pref := &Preference{
		Namespace: namespace,
		Key:       key,
	}

	if err := pref.SetValue(config); err != nil {
		return err
	}

	return s.PrefRepo.Save(ctx, pref)
}

type (
	Configurer interface {
		UnmarshalKey(name string, out interface{}) error
		Has(name string) bool
	}

	ConfigurerLoader struct {
		Service
		Cache      Cache
		Configurer Configurer
	}
)

func (s ConfigurerLoader) GetNamespaceValue(ctx context.Context, namespace string) (string, error) {
	value, err := s.Service.GetNamespaceValue(ctx, namespace)
	if err == nil {
		return value, nil
	}

	if !s.Configurer.Has(namespace) {
		return value, err
	}

	if err = s.Configurer.UnmarshalKey(namespace, &value); err != nil {
		return value, err
	}

	pref := &Preference{
		Namespace: namespaceSystem,
		Key:       namespace,
	}
	if err = pref.SetValue(value); err != nil {
		return value, nil
	}

	k := fmt.Sprintf("settings:pref:ns:key:%s:%s", pref.Namespace, pref.Key)
	t := fmt.Sprintf("settings:pref:tag:%s:%s", pref.Namespace, pref.Key)

	_ = s.Cache.Set(ctx, k, pref, t)

	return value, nil
}

func (s ConfigurerLoader) LoadConfigByNamespace(ctx context.Context, namespace string, config any) error {
	value, err := s.GetNamespaceValue(ctx, namespace)
	if err != nil {
		return err
	}

	return s.LoadConfig(ctx, value, config)
}

func (s ConfigurerLoader) LoadConfig(ctx context.Context, namespace string, config any) (err error) {
	if err = s.Service.LoadConfig(ctx, namespace, config); err == nil {
		return
	}

	if !s.Configurer.Has(namespace) {
		return
	}

	if err = s.Configurer.UnmarshalKey(namespace, config); err != nil {
		return
	}

	pref := &Preference{
		Namespace: namespace,
		Key:       keyConfiguration,
	}
	if err = pref.SetValue(config); err != nil {
		return nil
	}

	nk := "settings:ns:name:" + namespace
	nt := "settings:ns:tag:" + namespace

	_ = s.Cache.Set(ctx, nk, &Namespace{Name: namespace}, nt)

	k := fmt.Sprintf("settings:pref:ns:key:%s:%s", pref.Namespace, pref.Key)
	t := fmt.Sprintf("settings:pref:tag:%s:%s", pref.Namespace, pref.Key)

	_ = s.Cache.Set(ctx, k, pref, t)

	return
}
