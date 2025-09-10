package redisutil

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveSlots(t *testing.T) {
	type args struct {
		slots        []Slot
		removedSlots []Slot
	}
	tests := []struct {
		name string
		args args
		want []Slot
	}{
		{
			name: "1",
			args: args{
				slots:        []Slot{2, 3, 4, 5, 6, 7, 8, 9, 10},
				removedSlots: []Slot{2, 10},
			},
			want: []Slot{3, 4, 5, 6, 7, 8, 9},
		},
		{
			name: "2",
			args: args{
				slots:        []Slot{2, 5},
				removedSlots: []Slot{2, 2, 3},
			},
			want: []Slot{5},
		},
		{
			name: "3",
			args: args{
				slots:        []Slot{0, 1, 3, 4},
				removedSlots: []Slot{0, 1, 3, 4},
			},
			want: []Slot{},
		},
		{
			name: "4",
			args: args{
				slots:        []Slot{},
				removedSlots: []Slot{2, 10},
			},
			want: []Slot{},
		},
		{
			name: "5",
			args: args{
				slots:        []Slot{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				removedSlots: []Slot{5},
			},
			want: []Slot{0, 1, 2, 3, 4, 6, 7, 8, 9, 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveSlots(tt.args.slots, tt.args.removedSlots); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveSlots() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveSlot(t *testing.T) {
	type args struct {
		slots       []Slot
		removedSlot Slot
	}
	tests := []struct {
		name string
		args args
		want []Slot
	}{
		{
			name: "1",
			args: args{
				slots:       []Slot{2, 3, 4, 5, 6, 7, 8, 9, 10},
				removedSlot: 2,
			},
			want: []Slot{3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			name: "2",
			args: args{
				slots:       []Slot{2, 5},
				removedSlot: 2,
			},
			want: []Slot{5},
		},
		{
			name: "3",
			args: args{
				slots:       []Slot{0, 1, 3, 4},
				removedSlot: 3,
			},
			want: []Slot{0, 1, 4},
		},
		{
			name: "4",
			args: args{
				slots:       []Slot{},
				removedSlot: 2,
			},
			want: []Slot{},
		},
		{
			name: "5",
			args: args{
				slots:       []Slot{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
				removedSlot: 5,
			},
			want: []Slot{0, 1, 2, 3, 4, 6, 7, 8, 9, 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveSlot(tt.args.slots, tt.args.removedSlot); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveSlot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlot_ToInt(t *testing.T) {
	tests := []struct {
		name     string
		slot     Slot
		expected int
	}{
		{
			name:     "Slot 0",
			slot:     Slot(0),
			expected: 0,
		},
		{
			name:     "Slot 16383",
			slot:     Slot(16383),
			expected: 16383,
		},
		{
			name:     "Slot 1000",
			slot:     Slot(1000),
			expected: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.slot.ToInt()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildSlotSlice(t *testing.T) {
	tests := []struct {
		name     string
		start    Slot
		end      Slot
		expected []Slot
	}{
		{
			name:     "Single slot",
			start:    Slot(0),
			end:      Slot(0),
			expected: []Slot{0},
		},
		{
			name:     "Multiple slots",
			start:    Slot(0),
			end:      Slot(3),
			expected: []Slot{0, 1, 2, 3},
		},
		{
			name:     "Large range",
			start:    Slot(1000),
			end:      Slot(1002),
			expected: []Slot{1000, 1001, 1002},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSlotSlice(tt.start, tt.end)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddSlots(t *testing.T) {
	tests := []struct {
		name     string
		existing []Slot
		new      []Slot
		expected []Slot
	}{
		{
			name:     "Add to empty slice",
			existing: []Slot{},
			new:      []Slot{0, 1, 2},
			expected: []Slot{0, 1, 2},
		},
		{
			name:     "Add to existing slice",
			existing: []Slot{0, 1},
			new:      []Slot{2, 3},
			expected: []Slot{0, 1, 2, 3},
		},
		{
			name:     "Add empty slice",
			existing: []Slot{0, 1, 2},
			new:      []Slot{},
			expected: []Slot{0, 1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddSlots(tt.existing, tt.new)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSlotSlice(t *testing.T) {
	tests := []struct {
		name     string
		slots    []Slot
		expected interface{}
	}{
		{
			name:     "Empty slice",
			slots:    []Slot{},
			expected: "[]",
		},
		{
			name:     "Single slot",
			slots:    []Slot{0},
			expected: "[0-0]",
		},
		{
			name:     "Multiple slots",
			slots:    []Slot{0, 1, 2, 3},
			expected: "[0-3]",
		},
		{
			name:     "Large slots",
			slots:    []Slot{1000, 1001, 1002},
			expected: "[1000-1002]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SlotSlice(tt.slots).String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSlot_Swap tests the Swap method for Slots
func TestSlot_Swap(t *testing.T) {
	slots := SlotSlice{1, 2, 3, 4, 5}

	// Store original values
	originalSlot0 := slots[0]
	originalSlot1 := slots[1]

	// Swap slots
	slots.Swap(0, 1)

	// Verify swap
	if slots[0] != originalSlot1 {
		t.Errorf("Expected first slot to be %d, got %d", originalSlot1, slots[0])
	}
	if slots[1] != originalSlot0 {
		t.Errorf("Expected second slot to be %d, got %d", originalSlot0, slots[1])
	}
}

// TestSlot_Total tests the Total method
func TestSlot_Total(t *testing.T) {
	// Test with SlotRange since Total() is a method on SlotRange, not SlotSlice
	slotRange := &SlotRange{
		Min: 1,
		Max: 5,
	}

	total := slotRange.Total()
	expectedTotal := 5 // 5-1+1 = 5 slots

	if total != expectedTotal {
		t.Errorf("Expected total %d, got %d", expectedTotal, total)
	}

	// Test with single slot
	singleSlot := &SlotRange{
		Min: 42,
		Max: 42,
	}
	singleTotal := singleSlot.Total()
	if singleTotal != 1 {
		t.Errorf("Expected single total 1, got %d", singleTotal)
	}
}

// TestSlot_String tests the String method for Slots
func TestSlot_String(t *testing.T) {
	slots := SlotSlice{1, 2, 3, 4, 5}

	result := slots.String()
	expected := "[1-5]"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with empty slots
	emptySlots := SlotSlice{}
	emptyResult := emptySlots.String()
	emptyExpected := "[]"

	if emptyResult != emptyExpected {
		t.Errorf("Expected '%s', got '%s'", emptyExpected, emptyResult)
	}
}

// TestSlotRange_String tests the String method for SlotRange
func TestSlotRange_String(t *testing.T) {
	slotRange := &SlotRange{
		Min: 1,
		Max: 5,
	}

	result := slotRange.String()
	expected := "1-5"

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test with same start and end
	singleSlot := &SlotRange{
		Min: 42,
		Max: 42,
	}

	singleResult := singleSlot.String()
	singleExpected := "42-42"

	if singleResult != singleExpected {
		t.Errorf("Expected '%s', got '%s'", singleExpected, singleResult)
	}
}

// TestSlot_DecodeSlotRange tests the DecodeSlotRange function
func TestSlot_DecodeSlotRange(t *testing.T) {
	// Test valid slot range
	slots, importing, migrating, err := DecodeSlotRange("1-5")
	if err != nil {
		t.Errorf("Expected no error for '1-5', got %v", err)
	}
	if len(slots) != 5 {
		t.Errorf("Expected 5 slots, got %d", len(slots))
	}
	if importing != nil {
		t.Error("Expected no importing slot for range")
	}
	if migrating != nil {
		t.Error("Expected no migrating slot for range")
	}

	// Test single slot
	singleSlots, singleImporting, singleMigrating, err := DecodeSlotRange("42")
	if err != nil {
		t.Errorf("Expected no error for '42', got %v", err)
	}
	if len(singleSlots) != 1 {
		t.Errorf("Expected 1 slot, got %d", len(singleSlots))
	}
	if singleSlots[0] != 42 {
		t.Errorf("Expected slot 42, got %d", singleSlots[0])
	}
	if singleImporting != nil {
		t.Error("Expected no importing slot for single slot")
	}
	if singleMigrating != nil {
		t.Error("Expected no migrating slot for single slot")
	}

	// Test invalid format
	_, _, _, err = DecodeSlotRange("invalid")
	if err == nil {
		t.Error("Expected error for 'invalid' format")
	}

	// Test empty string
	_, _, _, err = DecodeSlotRange("")
	if err == nil {
		t.Error("Expected error for empty string")
	}

	// Test malformed range
	_, _, _, err = DecodeSlotRange("1-")
	if err == nil {
		t.Error("Expected error for malformed range '1-'")
	}

	_, _, _, err = DecodeSlotRange("-5")
	if err == nil {
		t.Error("Expected error for malformed range '-5'")
	}
}
