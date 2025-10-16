package pipeline

import (
	"testing"

	"github.com/SUNET/g119612/pkg/etsi119612"
)

func TestTSLTree(t *testing.T) {
	// Create test TSLs with references
	rootTSL := &etsi119612.TSL{Source: "root.xml"}
	refTSL1 := &etsi119612.TSL{Source: "ref1.xml"}
	refTSL2 := &etsi119612.TSL{Source: "ref2.xml"}
	refTSL3 := &etsi119612.TSL{Source: "ref3.xml"}

	// Set up the references
	rootTSL.Referenced = []*etsi119612.TSL{refTSL1, refTSL2}
	refTSL1.Referenced = []*etsi119612.TSL{refTSL3}

	// Create a tree from the root TSL
	tree := NewTSLTree(rootTSL)

	// Test that the tree was built correctly
	if tree.Root.TSL != rootTSL {
		t.Errorf("Root TSL not set correctly")
	}

	// Test child node count
	if len(tree.Root.Children) != 2 {
		t.Errorf("Root should have 2 children, got %d", len(tree.Root.Children))
	}

	// Test traverse function
	var visited []*etsi119612.TSL
	tree.Traverse(func(tsl *etsi119612.TSL) {
		visited = append(visited, tsl)
	})

	// Should have visited 4 TSLs in total
	if len(visited) != 4 {
		t.Errorf("Traverse should visit 4 TSLs, got %d", len(visited))
	}

	// Root TSL should be first in the traversal
	if visited[0] != rootTSL {
		t.Errorf("First TSL visited should be the root")
	}

	// Test finding a TSL by source
	found := tree.FindBySource("ref2.xml")
	if found != refTSL2 {
		t.Errorf("FindBySource failed to find ref2.xml")
	}

	// Test counting TSLs
	count := tree.Count()
	if count != 4 {
		t.Errorf("Count should return 4, got %d", count)
	}

	// Test converting to slice
	slice := tree.ToSlice()
	if len(slice) != 4 {
		t.Errorf("ToSlice should return 4 TSLs, got %d", len(slice))
	}

	// Test empty tree
	emptyTree := &TSLTree{}
	emptyCount := emptyTree.Count()
	if emptyCount != 0 {
		t.Errorf("Empty tree should have count 0, got %d", emptyCount)
	}
}

func TestTSLTreeInContext(t *testing.T) {
	// Create a context
	ctx := NewContext()

	// Ensure TSL trees stack is initialized
	ctx.EnsureTSLTrees()
	if ctx.TSLTrees == nil {
		t.Fatal("TSLTrees should be initialized")
	}

	// Test adding a TSL
	rootTSL := &etsi119612.TSL{Source: "root.xml"}
	refTSL := &etsi119612.TSL{Source: "ref.xml"}
	rootTSL.Referenced = []*etsi119612.TSL{refTSL}

	ctx.AddTSL(rootTSL)

	// Check that the tree was built and added to the stack
	tree, ok := ctx.TSLTrees.Peek()
	if !ok || tree == nil || tree.Root == nil || tree.Root.TSL != rootTSL {
		t.Fatal("TSLTree root was not set correctly")
	}

	// Test that copying preserves the tree
	newCtx := ctx.Copy()
	newTree, ok := newCtx.TSLTrees.Peek()
	if !ok || newTree == nil || newTree.Root == nil || newTree.Root.TSL != rootTSL {
		t.Fatal("TSLTree was not copied correctly")
	}

	// Test traversal in copied context
	var visited []*etsi119612.TSL
	newTree.Traverse(func(tsl *etsi119612.TSL) {
		visited = append(visited, tsl)
	})

	// Should have visited both TSLs
	if len(visited) != 2 {
		t.Errorf("Traverse should visit 2 TSLs, got %d", len(visited))
	}
}
