// Code generated by SQLBoiler 4.10.2 (https://github.com/volatiletech/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package model

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/friendsofgo/errors"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"github.com/volatiletech/sqlboiler/v4/queries/qmhelper"
	"github.com/volatiletech/strmangle"
)

// Purchase is an object representing the database table.
type Purchase struct {
	ID                  uint64      `boil:"id" json:"id" toml:"id" yaml:"id"`
	TransactionByHashID null.String `boil:"transaction_by_hash_id" json:"transaction_by_hash_id,omitempty" toml:"transaction_by_hash_id" yaml:"transaction_by_hash_id,omitempty"`
	Vout                uint        `boil:"vout" json:"vout" toml:"vout" yaml:"vout"`
	ClaimID             null.String `boil:"claim_id" json:"claim_id,omitempty" toml:"claim_id" yaml:"claim_id,omitempty"`
	PublisherID         null.String `boil:"publisher_id" json:"publisher_id,omitempty" toml:"publisher_id" yaml:"publisher_id,omitempty"`
	Height              uint        `boil:"height" json:"height" toml:"height" yaml:"height"`
	AmountSatoshi       int64       `boil:"amount_satoshi" json:"amount_satoshi" toml:"amount_satoshi" yaml:"amount_satoshi"`
	Created             time.Time   `boil:"created" json:"created" toml:"created" yaml:"created"`
	Modified            time.Time   `boil:"modified" json:"modified" toml:"modified" yaml:"modified"`

	R *purchaseR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L purchaseL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

var PurchaseColumns = struct {
	ID                  string
	TransactionByHashID string
	Vout                string
	ClaimID             string
	PublisherID         string
	Height              string
	AmountSatoshi       string
	Created             string
	Modified            string
}{
	ID:                  "id",
	TransactionByHashID: "transaction_by_hash_id",
	Vout:                "vout",
	ClaimID:             "claim_id",
	PublisherID:         "publisher_id",
	Height:              "height",
	AmountSatoshi:       "amount_satoshi",
	Created:             "created",
	Modified:            "modified",
}

var PurchaseTableColumns = struct {
	ID                  string
	TransactionByHashID string
	Vout                string
	ClaimID             string
	PublisherID         string
	Height              string
	AmountSatoshi       string
	Created             string
	Modified            string
}{
	ID:                  "purchase.id",
	TransactionByHashID: "purchase.transaction_by_hash_id",
	Vout:                "purchase.vout",
	ClaimID:             "purchase.claim_id",
	PublisherID:         "purchase.publisher_id",
	Height:              "purchase.height",
	AmountSatoshi:       "purchase.amount_satoshi",
	Created:             "purchase.created",
	Modified:            "purchase.modified",
}

// Generated where

var PurchaseWhere = struct {
	ID                  whereHelperuint64
	TransactionByHashID whereHelpernull_String
	Vout                whereHelperuint
	ClaimID             whereHelpernull_String
	PublisherID         whereHelpernull_String
	Height              whereHelperuint
	AmountSatoshi       whereHelperint64
	Created             whereHelpertime_Time
	Modified            whereHelpertime_Time
}{
	ID:                  whereHelperuint64{field: "`purchase`.`id`"},
	TransactionByHashID: whereHelpernull_String{field: "`purchase`.`transaction_by_hash_id`"},
	Vout:                whereHelperuint{field: "`purchase`.`vout`"},
	ClaimID:             whereHelpernull_String{field: "`purchase`.`claim_id`"},
	PublisherID:         whereHelpernull_String{field: "`purchase`.`publisher_id`"},
	Height:              whereHelperuint{field: "`purchase`.`height`"},
	AmountSatoshi:       whereHelperint64{field: "`purchase`.`amount_satoshi`"},
	Created:             whereHelpertime_Time{field: "`purchase`.`created`"},
	Modified:            whereHelpertime_Time{field: "`purchase`.`modified`"},
}

// PurchaseRels is where relationship names are stored.
var PurchaseRels = struct {
	TransactionByHash string
}{
	TransactionByHash: "TransactionByHash",
}

// purchaseR is where relationships are stored.
type purchaseR struct {
	TransactionByHash *Transaction `boil:"TransactionByHash" json:"TransactionByHash" toml:"TransactionByHash" yaml:"TransactionByHash"`
}

// NewStruct creates a new relationship struct
func (*purchaseR) NewStruct() *purchaseR {
	return &purchaseR{}
}

// purchaseL is where Load methods for each relationship are stored.
type purchaseL struct{}

var (
	purchaseAllColumns            = []string{"id", "transaction_by_hash_id", "vout", "claim_id", "publisher_id", "height", "amount_satoshi", "created", "modified"}
	purchaseColumnsWithoutDefault = []string{"transaction_by_hash_id", "vout", "claim_id", "publisher_id", "height"}
	purchaseColumnsWithDefault    = []string{"id", "amount_satoshi", "created", "modified"}
	purchasePrimaryKeyColumns     = []string{"id"}
	purchaseGeneratedColumns      = []string{}
)

type (
	// PurchaseSlice is an alias for a slice of pointers to Purchase.
	// This should almost always be used instead of []Purchase.
	PurchaseSlice []*Purchase

	purchaseQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	purchaseType                 = reflect.TypeOf(&Purchase{})
	purchaseMapping              = queries.MakeStructMapping(purchaseType)
	purchasePrimaryKeyMapping, _ = queries.BindMapping(purchaseType, purchaseMapping, purchasePrimaryKeyColumns)
	purchaseInsertCacheMut       sync.RWMutex
	purchaseInsertCache          = make(map[string]insertCache)
	purchaseUpdateCacheMut       sync.RWMutex
	purchaseUpdateCache          = make(map[string]updateCache)
	purchaseUpsertCacheMut       sync.RWMutex
	purchaseUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force qmhelper dependency for where clause generation (which doesn't
	// always happen)
	_ = qmhelper.Where
)

// OneG returns a single purchase record from the query using the global executor.
func (q purchaseQuery) OneG() (*Purchase, error) {
	return q.One(boil.GetDB())
}

// OneGP returns a single purchase record from the query using the global executor, and panics on error.
func (q purchaseQuery) OneGP() *Purchase {
	o, err := q.One(boil.GetDB())
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return o
}

// OneP returns a single purchase record from the query, and panics on error.
func (q purchaseQuery) OneP(exec boil.Executor) *Purchase {
	o, err := q.One(exec)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return o
}

// One returns a single purchase record from the query.
func (q purchaseQuery) One(exec boil.Executor) (*Purchase, error) {
	o := &Purchase{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(nil, exec, o)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "model: failed to execute a one query for purchase")
	}

	return o, nil
}

// AllG returns all Purchase records from the query using the global executor.
func (q purchaseQuery) AllG() (PurchaseSlice, error) {
	return q.All(boil.GetDB())
}

// AllGP returns all Purchase records from the query using the global executor, and panics on error.
func (q purchaseQuery) AllGP() PurchaseSlice {
	o, err := q.All(boil.GetDB())
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return o
}

// AllP returns all Purchase records from the query, and panics on error.
func (q purchaseQuery) AllP(exec boil.Executor) PurchaseSlice {
	o, err := q.All(exec)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return o
}

// All returns all Purchase records from the query.
func (q purchaseQuery) All(exec boil.Executor) (PurchaseSlice, error) {
	var o []*Purchase

	err := q.Bind(nil, exec, &o)
	if err != nil {
		return nil, errors.Wrap(err, "model: failed to assign all query results to Purchase slice")
	}

	return o, nil
}

// CountG returns the count of all Purchase records in the query using the global executor
func (q purchaseQuery) CountG() (int64, error) {
	return q.Count(boil.GetDB())
}

// CountGP returns the count of all Purchase records in the query using the global executor, and panics on error.
func (q purchaseQuery) CountGP() int64 {
	c, err := q.Count(boil.GetDB())
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return c
}

// CountP returns the count of all Purchase records in the query, and panics on error.
func (q purchaseQuery) CountP(exec boil.Executor) int64 {
	c, err := q.Count(exec)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return c
}

// Count returns the count of all Purchase records in the query.
func (q purchaseQuery) Count(exec boil.Executor) (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRow(exec).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "model: failed to count purchase rows")
	}

	return count, nil
}

// ExistsG checks if the row exists in the table using the global executor.
func (q purchaseQuery) ExistsG() (bool, error) {
	return q.Exists(boil.GetDB())
}

// ExistsGP checks if the row exists in the table using the global executor, and panics on error.
func (q purchaseQuery) ExistsGP() bool {
	e, err := q.Exists(boil.GetDB())
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}

// ExistsP checks if the row exists in the table, and panics on error.
func (q purchaseQuery) ExistsP(exec boil.Executor) bool {
	e, err := q.Exists(exec)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}

// Exists checks if the row exists in the table.
func (q purchaseQuery) Exists(exec boil.Executor) (bool, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRow(exec).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "model: failed to check if purchase exists")
	}

	return count > 0, nil
}

// TransactionByHash pointed to by the foreign key.
func (o *Purchase) TransactionByHash(mods ...qm.QueryMod) transactionQuery {
	queryMods := []qm.QueryMod{
		qm.Where("`hash` = ?", o.TransactionByHashID),
	}

	queryMods = append(queryMods, mods...)

	return Transactions(queryMods...)
}

// LoadTransactionByHash allows an eager lookup of values, cached into the
// loaded structs of the objects. This is for an N-1 relationship.
func (purchaseL) LoadTransactionByHash(e boil.Executor, singular bool, maybePurchase interface{}, mods queries.Applicator) error {
	var slice []*Purchase
	var object *Purchase

	if singular {
		object = maybePurchase.(*Purchase)
	} else {
		slice = *maybePurchase.(*[]*Purchase)
	}

	args := make([]interface{}, 0, 1)
	if singular {
		if object.R == nil {
			object.R = &purchaseR{}
		}
		if !queries.IsNil(object.TransactionByHashID) {
			args = append(args, object.TransactionByHashID)
		}

	} else {
	Outer:
		for _, obj := range slice {
			if obj.R == nil {
				obj.R = &purchaseR{}
			}

			for _, a := range args {
				if queries.Equal(a, obj.TransactionByHashID) {
					continue Outer
				}
			}

			if !queries.IsNil(obj.TransactionByHashID) {
				args = append(args, obj.TransactionByHashID)
			}

		}
	}

	if len(args) == 0 {
		return nil
	}

	query := NewQuery(
		qm.From(`transaction`),
		qm.WhereIn(`transaction.hash in ?`, args...),
	)
	if mods != nil {
		mods.Apply(query)
	}

	results, err := query.Query(e)
	if err != nil {
		return errors.Wrap(err, "failed to eager load Transaction")
	}

	var resultSlice []*Transaction
	if err = queries.Bind(results, &resultSlice); err != nil {
		return errors.Wrap(err, "failed to bind eager loaded slice Transaction")
	}

	if err = results.Close(); err != nil {
		return errors.Wrap(err, "failed to close results of eager load for transaction")
	}
	if err = results.Err(); err != nil {
		return errors.Wrap(err, "error occurred during iteration of eager loaded relations for transaction")
	}

	if len(resultSlice) == 0 {
		return nil
	}

	if singular {
		foreign := resultSlice[0]
		object.R.TransactionByHash = foreign
		if foreign.R == nil {
			foreign.R = &transactionR{}
		}
		foreign.R.TransactionByHashPurchases = append(foreign.R.TransactionByHashPurchases, object)
		return nil
	}

	for _, local := range slice {
		for _, foreign := range resultSlice {
			if queries.Equal(local.TransactionByHashID, foreign.Hash) {
				local.R.TransactionByHash = foreign
				if foreign.R == nil {
					foreign.R = &transactionR{}
				}
				foreign.R.TransactionByHashPurchases = append(foreign.R.TransactionByHashPurchases, local)
				break
			}
		}
	}

	return nil
}

// SetTransactionByHashG of the purchase to the related item.
// Sets o.R.TransactionByHash to related.
// Adds o to related.R.TransactionByHashPurchases.
// Uses the global database handle.
func (o *Purchase) SetTransactionByHashG(insert bool, related *Transaction) error {
	return o.SetTransactionByHash(boil.GetDB(), insert, related)
}

// SetTransactionByHashP of the purchase to the related item.
// Sets o.R.TransactionByHash to related.
// Adds o to related.R.TransactionByHashPurchases.
// Panics on error.
func (o *Purchase) SetTransactionByHashP(exec boil.Executor, insert bool, related *Transaction) {
	if err := o.SetTransactionByHash(exec, insert, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetTransactionByHashGP of the purchase to the related item.
// Sets o.R.TransactionByHash to related.
// Adds o to related.R.TransactionByHashPurchases.
// Uses the global database handle and panics on error.
func (o *Purchase) SetTransactionByHashGP(insert bool, related *Transaction) {
	if err := o.SetTransactionByHash(boil.GetDB(), insert, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// SetTransactionByHash of the purchase to the related item.
// Sets o.R.TransactionByHash to related.
// Adds o to related.R.TransactionByHashPurchases.
func (o *Purchase) SetTransactionByHash(exec boil.Executor, insert bool, related *Transaction) error {
	var err error
	if insert {
		if err = related.Insert(exec, boil.Infer()); err != nil {
			return errors.Wrap(err, "failed to insert into foreign table")
		}
	}

	updateQuery := fmt.Sprintf(
		"UPDATE `purchase` SET %s WHERE %s",
		strmangle.SetParamNames("`", "`", 0, []string{"transaction_by_hash_id"}),
		strmangle.WhereClause("`", "`", 0, purchasePrimaryKeyColumns),
	)
	values := []interface{}{related.Hash, o.ID}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, updateQuery)
		fmt.Fprintln(boil.DebugWriter, values)
	}
	if _, err = exec.Exec(updateQuery, values...); err != nil {
		return errors.Wrap(err, "failed to update local table")
	}

	queries.Assign(&o.TransactionByHashID, related.Hash)
	if o.R == nil {
		o.R = &purchaseR{
			TransactionByHash: related,
		}
	} else {
		o.R.TransactionByHash = related
	}

	if related.R == nil {
		related.R = &transactionR{
			TransactionByHashPurchases: PurchaseSlice{o},
		}
	} else {
		related.R.TransactionByHashPurchases = append(related.R.TransactionByHashPurchases, o)
	}

	return nil
}

// RemoveTransactionByHashG relationship.
// Sets o.R.TransactionByHash to nil.
// Removes o from all passed in related items' relationships struct.
// Uses the global database handle.
func (o *Purchase) RemoveTransactionByHashG(related *Transaction) error {
	return o.RemoveTransactionByHash(boil.GetDB(), related)
}

// RemoveTransactionByHashP relationship.
// Sets o.R.TransactionByHash to nil.
// Removes o from all passed in related items' relationships struct.
// Panics on error.
func (o *Purchase) RemoveTransactionByHashP(exec boil.Executor, related *Transaction) {
	if err := o.RemoveTransactionByHash(exec, related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveTransactionByHashGP relationship.
// Sets o.R.TransactionByHash to nil.
// Removes o from all passed in related items' relationships struct.
// Uses the global database handle and panics on error.
func (o *Purchase) RemoveTransactionByHashGP(related *Transaction) {
	if err := o.RemoveTransactionByHash(boil.GetDB(), related); err != nil {
		panic(boil.WrapErr(err))
	}
}

// RemoveTransactionByHash relationship.
// Sets o.R.TransactionByHash to nil.
// Removes o from all passed in related items' relationships struct.
func (o *Purchase) RemoveTransactionByHash(exec boil.Executor, related *Transaction) error {
	var err error

	queries.SetScanner(&o.TransactionByHashID, nil)
	if err = o.Update(exec, boil.Whitelist("transaction_by_hash_id")); err != nil {
		return errors.Wrap(err, "failed to update local table")
	}

	if o.R != nil {
		o.R.TransactionByHash = nil
	}
	if related == nil || related.R == nil {
		return nil
	}

	for i, ri := range related.R.TransactionByHashPurchases {
		if queries.Equal(o.TransactionByHashID, ri.TransactionByHashID) {
			continue
		}

		ln := len(related.R.TransactionByHashPurchases)
		if ln > 1 && i < ln-1 {
			related.R.TransactionByHashPurchases[i] = related.R.TransactionByHashPurchases[ln-1]
		}
		related.R.TransactionByHashPurchases = related.R.TransactionByHashPurchases[:ln-1]
		break
	}
	return nil
}

// Purchases retrieves all the records using an executor.
func Purchases(mods ...qm.QueryMod) purchaseQuery {
	mods = append(mods, qm.From("`purchase`"))
	q := NewQuery(mods...)
	if len(queries.GetSelect(q)) == 0 {
		queries.SetSelect(q, []string{"`purchase`.*"})
	}

	return purchaseQuery{q}
}

// FindPurchaseG retrieves a single record by ID.
func FindPurchaseG(iD uint64, selectCols ...string) (*Purchase, error) {
	return FindPurchase(boil.GetDB(), iD, selectCols...)
}

// FindPurchaseP retrieves a single record by ID with an executor, and panics on error.
func FindPurchaseP(exec boil.Executor, iD uint64, selectCols ...string) *Purchase {
	retobj, err := FindPurchase(exec, iD, selectCols...)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return retobj
}

// FindPurchaseGP retrieves a single record by ID, and panics on error.
func FindPurchaseGP(iD uint64, selectCols ...string) *Purchase {
	retobj, err := FindPurchase(boil.GetDB(), iD, selectCols...)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return retobj
}

// FindPurchase retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindPurchase(exec boil.Executor, iD uint64, selectCols ...string) (*Purchase, error) {
	purchaseObj := &Purchase{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from `purchase` where `id`=?", sel,
	)

	q := queries.Raw(query, iD)

	err := q.Bind(nil, exec, purchaseObj)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "model: unable to select from purchase")
	}

	return purchaseObj, nil
}

// InsertG a single record. See Insert for whitelist behavior description.
func (o *Purchase) InsertG(columns boil.Columns) error {
	return o.Insert(boil.GetDB(), columns)
}

// InsertP a single record using an executor, and panics on error. See Insert
// for whitelist behavior description.
func (o *Purchase) InsertP(exec boil.Executor, columns boil.Columns) {
	if err := o.Insert(exec, columns); err != nil {
		panic(boil.WrapErr(err))
	}
}

// InsertGP a single record, and panics on error. See Insert for whitelist
// behavior description.
func (o *Purchase) InsertGP(columns boil.Columns) {
	if err := o.Insert(boil.GetDB(), columns); err != nil {
		panic(boil.WrapErr(err))
	}
}

// Insert a single record using an executor.
// See boil.Columns.InsertColumnSet documentation to understand column list inference for inserts.
func (o *Purchase) Insert(exec boil.Executor, columns boil.Columns) error {
	if o == nil {
		return errors.New("model: no purchase provided for insertion")
	}

	var err error

	nzDefaults := queries.NonZeroDefaultSet(purchaseColumnsWithDefault, o)

	key := makeCacheKey(columns, nzDefaults)
	purchaseInsertCacheMut.RLock()
	cache, cached := purchaseInsertCache[key]
	purchaseInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := columns.InsertColumnSet(
			purchaseAllColumns,
			purchaseColumnsWithDefault,
			purchaseColumnsWithoutDefault,
			nzDefaults,
		)

		cache.valueMapping, err = queries.BindMapping(purchaseType, purchaseMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(purchaseType, purchaseMapping, returnColumns)
		if err != nil {
			return err
		}
		if len(wl) != 0 {
			cache.query = fmt.Sprintf("INSERT INTO `purchase` (`%s`) %%sVALUES (%s)%%s", strings.Join(wl, "`,`"), strmangle.Placeholders(dialect.UseIndexPlaceholders, len(wl), 1, 1))
		} else {
			cache.query = "INSERT INTO `purchase` () VALUES ()%s%s"
		}

		var queryOutput, queryReturning string

		if len(cache.retMapping) != 0 {
			cache.retQuery = fmt.Sprintf("SELECT `%s` FROM `purchase` WHERE %s", strings.Join(returnColumns, "`,`"), strmangle.WhereClause("`", "`", 0, purchasePrimaryKeyColumns))
		}

		cache.query = fmt.Sprintf(cache.query, queryOutput, queryReturning)
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, vals)
	}
	result, err := exec.Exec(cache.query, vals...)

	if err != nil {
		return errors.Wrap(err, "model: unable to insert into purchase")
	}

	var lastID int64
	var identifierCols []interface{}

	if len(cache.retMapping) == 0 {
		goto CacheNoHooks
	}

	lastID, err = result.LastInsertId()
	if err != nil {
		return ErrSyncFail
	}

	o.ID = uint64(lastID)
	if lastID != 0 && len(cache.retMapping) == 1 && cache.retMapping[0] == purchaseMapping["id"] {
		goto CacheNoHooks
	}

	identifierCols = []interface{}{
		o.ID,
	}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.retQuery)
		fmt.Fprintln(boil.DebugWriter, identifierCols...)
	}
	err = exec.QueryRow(cache.retQuery, identifierCols...).Scan(queries.PtrsFromMapping(value, cache.retMapping)...)
	if err != nil {
		return errors.Wrap(err, "model: unable to populate default values for purchase")
	}

CacheNoHooks:
	if !cached {
		purchaseInsertCacheMut.Lock()
		purchaseInsertCache[key] = cache
		purchaseInsertCacheMut.Unlock()
	}

	return nil
}

// UpdateG a single Purchase record using the global executor.
// See Update for more documentation.
func (o *Purchase) UpdateG(columns boil.Columns) error {
	return o.Update(boil.GetDB(), columns)
}

// UpdateP uses an executor to update the Purchase, and panics on error.
// See Update for more documentation.
func (o *Purchase) UpdateP(exec boil.Executor, columns boil.Columns) {
	err := o.Update(exec, columns)
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateGP a single Purchase record using the global executor. Panics on error.
// See Update for more documentation.
func (o *Purchase) UpdateGP(columns boil.Columns) {
	err := o.Update(boil.GetDB(), columns)
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// Update uses an executor to update the Purchase.
// See boil.Columns.UpdateColumnSet documentation to understand column list inference for updates.
// Update does not automatically update the record in case of default values. Use .Reload() to refresh the records.
func (o *Purchase) Update(exec boil.Executor, columns boil.Columns) error {
	var err error
	key := makeCacheKey(columns, nil)
	purchaseUpdateCacheMut.RLock()
	cache, cached := purchaseUpdateCache[key]
	purchaseUpdateCacheMut.RUnlock()

	if !cached {
		wl := columns.UpdateColumnSet(
			purchaseAllColumns,
			purchasePrimaryKeyColumns,
		)
		if len(wl) == 0 {
			return errors.New("model: unable to update purchase, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE `purchase` SET %s WHERE %s",
			strmangle.SetParamNames("`", "`", 0, wl),
			strmangle.WhereClause("`", "`", 0, purchasePrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(purchaseType, purchaseMapping, append(wl, purchasePrimaryKeyColumns...))
		if err != nil {
			return err
		}
	}

	values := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), cache.valueMapping)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, values)
	}
	_, err = exec.Exec(cache.query, values...)
	if err != nil {
		return errors.Wrap(err, "model: unable to update purchase row")
	}

	if !cached {
		purchaseUpdateCacheMut.Lock()
		purchaseUpdateCache[key] = cache
		purchaseUpdateCacheMut.Unlock()
	}

	return nil
}

// UpdateAllP updates all rows with matching column names, and panics on error.
func (q purchaseQuery) UpdateAllP(exec boil.Executor, cols M) {
	err := q.UpdateAll(exec, cols)
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAllG updates all rows with the specified column values.
func (q purchaseQuery) UpdateAllG(cols M) error {
	return q.UpdateAll(boil.GetDB(), cols)
}

// UpdateAllGP updates all rows with the specified column values, and panics on error.
func (q purchaseQuery) UpdateAllGP(cols M) {
	err := q.UpdateAll(boil.GetDB(), cols)
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAll updates all rows with the specified column values.
func (q purchaseQuery) UpdateAll(exec boil.Executor, cols M) error {
	queries.SetUpdate(q.Query, cols)

	_, err := q.Query.Exec(exec)
	if err != nil {
		return errors.Wrap(err, "model: unable to update all for purchase")
	}

	return nil
}

// UpdateAllG updates all rows with the specified column values.
func (o PurchaseSlice) UpdateAllG(cols M) error {
	return o.UpdateAll(boil.GetDB(), cols)
}

// UpdateAllGP updates all rows with the specified column values, and panics on error.
func (o PurchaseSlice) UpdateAllGP(cols M) {
	err := o.UpdateAll(boil.GetDB(), cols)
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAllP updates all rows with the specified column values, and panics on error.
func (o PurchaseSlice) UpdateAllP(exec boil.Executor, cols M) {
	err := o.UpdateAll(exec, cols)
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o PurchaseSlice) UpdateAll(exec boil.Executor, cols M) error {
	ln := int64(len(o))
	if ln == 0 {
		return nil
	}

	if len(cols) == 0 {
		return errors.New("model: update all requires at least one column argument")
	}

	colNames := make([]string, len(cols))
	args := make([]interface{}, len(cols))

	i := 0
	for name, value := range cols {
		colNames[i] = name
		args[i] = value
		i++
	}

	// Append all of the primary key values for each column
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), purchasePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf("UPDATE `purchase` SET %s WHERE %s",
		strmangle.SetParamNames("`", "`", 0, colNames),
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 0, purchasePrimaryKeyColumns, len(o)))

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}
	_, err := exec.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "model: unable to update all in purchase slice")
	}

	return nil
}

// UpsertG attempts an insert, and does an update or ignore on conflict.
func (o *Purchase) UpsertG(updateColumns, insertColumns boil.Columns) error {
	return o.Upsert(boil.GetDB(), updateColumns, insertColumns)
}

// UpsertGP attempts an insert, and does an update or ignore on conflict. Panics on error.
func (o *Purchase) UpsertGP(updateColumns, insertColumns boil.Columns) {
	if err := o.Upsert(boil.GetDB(), updateColumns, insertColumns); err != nil {
		panic(boil.WrapErr(err))
	}
}

// UpsertP attempts an insert using an executor, and does an update or ignore on conflict.
// UpsertP panics on error.
func (o *Purchase) UpsertP(exec boil.Executor, updateColumns, insertColumns boil.Columns) {
	if err := o.Upsert(exec, updateColumns, insertColumns); err != nil {
		panic(boil.WrapErr(err))
	}
}

var mySQLPurchaseUniqueColumns = []string{
	"id",
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
// See boil.Columns documentation for how to properly use updateColumns and insertColumns.
func (o *Purchase) Upsert(exec boil.Executor, updateColumns, insertColumns boil.Columns) error {
	if o == nil {
		return errors.New("model: no purchase provided for upsert")
	}

	nzDefaults := queries.NonZeroDefaultSet(purchaseColumnsWithDefault, o)
	nzUniques := queries.NonZeroDefaultSet(mySQLPurchaseUniqueColumns, o)

	if len(nzUniques) == 0 {
		return errors.New("cannot upsert with a table that cannot conflict on a unique column")
	}

	// Build cache key in-line uglily - mysql vs psql problems
	buf := strmangle.GetBuffer()
	buf.WriteString(strconv.Itoa(updateColumns.Kind))
	for _, c := range updateColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(insertColumns.Kind))
	for _, c := range insertColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	for _, c := range nzDefaults {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	for _, c := range nzUniques {
		buf.WriteString(c)
	}
	key := buf.String()
	strmangle.PutBuffer(buf)

	purchaseUpsertCacheMut.RLock()
	cache, cached := purchaseUpsertCache[key]
	purchaseUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		insert, ret := insertColumns.InsertColumnSet(
			purchaseAllColumns,
			purchaseColumnsWithDefault,
			purchaseColumnsWithoutDefault,
			nzDefaults,
		)

		update := updateColumns.UpdateColumnSet(
			purchaseAllColumns,
			purchasePrimaryKeyColumns,
		)

		if !updateColumns.IsNone() && len(update) == 0 {
			return errors.New("model: unable to upsert purchase, could not build update column list")
		}

		ret = strmangle.SetComplement(ret, nzUniques)
		cache.query = buildUpsertQueryMySQL(dialect, "`purchase`", update, insert)
		cache.retQuery = fmt.Sprintf(
			"SELECT %s FROM `purchase` WHERE %s",
			strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, ret), ","),
			strmangle.WhereClause("`", "`", 0, nzUniques),
		)

		cache.valueMapping, err = queries.BindMapping(purchaseType, purchaseMapping, insert)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(purchaseType, purchaseMapping, ret)
			if err != nil {
				return err
			}
		}
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)
	var returns []interface{}
	if len(cache.retMapping) != 0 {
		returns = queries.PtrsFromMapping(value, cache.retMapping)
	}

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.query)
		fmt.Fprintln(boil.DebugWriter, vals)
	}
	result, err := exec.Exec(cache.query, vals...)

	if err != nil {
		return errors.Wrap(err, "model: unable to upsert for purchase")
	}

	var lastID int64
	var uniqueMap []uint64
	var nzUniqueCols []interface{}

	if len(cache.retMapping) == 0 {
		goto CacheNoHooks
	}

	lastID, err = result.LastInsertId()
	if err != nil {
		return ErrSyncFail
	}

	o.ID = uint64(lastID)
	if lastID != 0 && len(cache.retMapping) == 1 && cache.retMapping[0] == purchaseMapping["id"] {
		goto CacheNoHooks
	}

	uniqueMap, err = queries.BindMapping(purchaseType, purchaseMapping, nzUniques)
	if err != nil {
		return errors.Wrap(err, "model: unable to retrieve unique values for purchase")
	}
	nzUniqueCols = queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), uniqueMap)

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, cache.retQuery)
		fmt.Fprintln(boil.DebugWriter, nzUniqueCols...)
	}
	err = exec.QueryRow(cache.retQuery, nzUniqueCols...).Scan(returns...)
	if err != nil {
		return errors.Wrap(err, "model: unable to populate default values for purchase")
	}

CacheNoHooks:
	if !cached {
		purchaseUpsertCacheMut.Lock()
		purchaseUpsertCache[key] = cache
		purchaseUpsertCacheMut.Unlock()
	}

	return nil
}

// DeleteG deletes a single Purchase record.
// DeleteG will match against the primary key column to find the record to delete.
func (o *Purchase) DeleteG() error {
	return o.Delete(boil.GetDB())
}

// DeleteP deletes a single Purchase record with an executor.
// DeleteP will match against the primary key column to find the record to delete.
// Panics on error.
func (o *Purchase) DeleteP(exec boil.Executor) {
	err := o.Delete(exec)
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteGP deletes a single Purchase record.
// DeleteGP will match against the primary key column to find the record to delete.
// Panics on error.
func (o *Purchase) DeleteGP() {
	err := o.Delete(boil.GetDB())
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// Delete deletes a single Purchase record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *Purchase) Delete(exec boil.Executor) error {
	if o == nil {
		return errors.New("model: no Purchase provided for delete")
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), purchasePrimaryKeyMapping)
	sql := "DELETE FROM `purchase` WHERE `id`=?"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args...)
	}
	_, err := exec.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "model: unable to delete from purchase")
	}

	return nil
}

func (q purchaseQuery) DeleteAllG() error {
	return q.DeleteAll(boil.GetDB())
}

// DeleteAllP deletes all rows, and panics on error.
func (q purchaseQuery) DeleteAllP(exec boil.Executor) {
	err := q.DeleteAll(exec)
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAllGP deletes all rows, and panics on error.
func (q purchaseQuery) DeleteAllGP() {
	err := q.DeleteAll(boil.GetDB())
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAll deletes all matching rows.
func (q purchaseQuery) DeleteAll(exec boil.Executor) error {
	if q.Query == nil {
		return errors.New("model: no purchaseQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	_, err := q.Query.Exec(exec)
	if err != nil {
		return errors.Wrap(err, "model: unable to delete all from purchase")
	}

	return nil
}

// DeleteAllG deletes all rows in the slice.
func (o PurchaseSlice) DeleteAllG() error {
	return o.DeleteAll(boil.GetDB())
}

// DeleteAllP deletes all rows in the slice, using an executor, and panics on error.
func (o PurchaseSlice) DeleteAllP(exec boil.Executor) {
	err := o.DeleteAll(exec)
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAllGP deletes all rows in the slice, and panics on error.
func (o PurchaseSlice) DeleteAllGP() {
	err := o.DeleteAll(boil.GetDB())
	if err != nil {
		panic(boil.WrapErr(err))
	}
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o PurchaseSlice) DeleteAll(exec boil.Executor) error {
	if len(o) == 0 {
		return nil
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), purchasePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "DELETE FROM `purchase` WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 0, purchasePrimaryKeyColumns, len(o))

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, args)
	}
	_, err := exec.Exec(sql, args...)
	if err != nil {
		return errors.Wrap(err, "model: unable to delete all from purchase slice")
	}

	return nil
}

// ReloadG refetches the object from the database using the primary keys.
func (o *Purchase) ReloadG() error {
	if o == nil {
		return errors.New("model: no Purchase provided for reload")
	}

	return o.Reload(boil.GetDB())
}

// ReloadP refetches the object from the database with an executor. Panics on error.
func (o *Purchase) ReloadP(exec boil.Executor) {
	if err := o.Reload(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadGP refetches the object from the database and panics on error.
func (o *Purchase) ReloadGP() {
	if err := o.Reload(boil.GetDB()); err != nil {
		panic(boil.WrapErr(err))
	}
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *Purchase) Reload(exec boil.Executor) error {
	ret, err := FindPurchase(exec, o.ID)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAllG refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *PurchaseSlice) ReloadAllG() error {
	if o == nil {
		return errors.New("model: empty PurchaseSlice provided for reload all")
	}

	return o.ReloadAll(boil.GetDB())
}

// ReloadAllP refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
// Panics on error.
func (o *PurchaseSlice) ReloadAllP(exec boil.Executor) {
	if err := o.ReloadAll(exec); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadAllGP refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
// Panics on error.
func (o *PurchaseSlice) ReloadAllGP() {
	if err := o.ReloadAll(boil.GetDB()); err != nil {
		panic(boil.WrapErr(err))
	}
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *PurchaseSlice) ReloadAll(exec boil.Executor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	slice := PurchaseSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), purchasePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "SELECT `purchase`.* FROM `purchase` WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 0, purchasePrimaryKeyColumns, len(*o))

	q := queries.Raw(sql, args...)

	err := q.Bind(nil, exec, &slice)
	if err != nil {
		return errors.Wrap(err, "model: unable to reload all in PurchaseSlice")
	}

	*o = slice

	return nil
}

// PurchaseExistsG checks if the Purchase row exists.
func PurchaseExistsG(iD uint64) (bool, error) {
	return PurchaseExists(boil.GetDB(), iD)
}

// PurchaseExistsP checks if the Purchase row exists. Panics on error.
func PurchaseExistsP(exec boil.Executor, iD uint64) bool {
	e, err := PurchaseExists(exec, iD)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}

// PurchaseExistsGP checks if the Purchase row exists. Panics on error.
func PurchaseExistsGP(iD uint64) bool {
	e, err := PurchaseExists(boil.GetDB(), iD)
	if err != nil {
		panic(boil.WrapErr(err))
	}

	return e
}

// PurchaseExists checks if the Purchase row exists.
func PurchaseExists(exec boil.Executor, iD uint64) (bool, error) {
	var exists bool
	sql := "select exists(select 1 from `purchase` where `id`=? limit 1)"

	if boil.DebugMode {
		fmt.Fprintln(boil.DebugWriter, sql)
		fmt.Fprintln(boil.DebugWriter, iD)
	}
	row := exec.QueryRow(sql, iD)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "model: unable to check if purchase exists")
	}

	return exists, nil
}
