package json2go

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"
)

func astMakeDecls(rootNodes []*node, opts options) []ast.Decl {
	var decls []ast.Decl

	for _, node := range rootNodes {
		decls = append(decls, &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{
					Name: ast.NewIdent(node.name),
					Type: astTypeFromNode(node, opts),
				},
			},
		})
	}

	return decls
}

func astTypeFromNode(n *node, opts options) ast.Expr {
	var resultType ast.Expr
	notRequiredAsPointer := true
	allowPointer := true

	switch n.t.(type) {
	case nodeBoolType:
		resultType = ast.NewIdent("bool")
	case nodeIntType:
		resultType = ast.NewIdent("int")
	case nodeFloatType:
		resultType = ast.NewIdent("float64")
	case nodeStringType:
		resultType = ast.NewIdent("string")
		notRequiredAsPointer = opts.stringPointersWhenKeyMissing
	case nodeTimeType:
		if opts.timeAsStr {
			resultType = ast.NewIdent("string")
			notRequiredAsPointer = opts.stringPointersWhenKeyMissing
		} else {
			resultType = ast.NewIdent("time.Time")
		}
	case nodeObjectType:
		resultType = astStructTypeFromNode(n, opts)
	case nodeExtractedType:
		extName := n.externalTypeID
		if extName == "" {
			extName = n.name
		}
		resultType = ast.NewIdent(extName)
	case nodeInterfaceType, nodeInitType:
		resultType = newEmptyInterfaceExpr()
		allowPointer = false
	case nodeMapType:
		var ve ast.Expr
		if len(n.children) == 0 {
			ve = newEmptyInterfaceExpr()
		} else {
			ve = astTypeFromNode(n.children[0], opts)
		}
		resultType = &ast.MapType{
			Key:   ast.NewIdent("string"),
			Value: ve,
		}
		allowPointer = false
	case nodeOtherType:
		resultType = ast.NewIdent(n.t.id())
		allowPointer = false
	default:
		panic(fmt.Sprintf("unknown type: %v", n.t))
	}

	if !n.root && n.arrayLevel == 0 && allowPointer {
		if n.nullable || (!n.required && notRequiredAsPointer) {
			resultType = &ast.StarExpr{
				X: resultType,
			}
		}
	}

	for i := n.arrayLevel; i > 0; i-- {
		resultType = &ast.ArrayType{
			Elt: resultType,
		}
	}

	return resultType
}

func astStructTypeFromNode(n *node, opts options) *ast.StructType {
	typeDesc := &ast.StructType{
		Fields: &ast.FieldList{
			List: []*ast.Field{},
		},
	}

	// sort children by name
	type nodeWithName struct {
		name string
		node *node
	}
	var sortedChildren []nodeWithName
	for _, child := range n.children {
		sortedChildren = append(sortedChildren, nodeWithName{
			name: child.name,
			node: child,
		})
	}
	sort.Slice(sortedChildren, func(i, j int) bool {
		return sortedChildren[i].name < sortedChildren[j].name
	})

	for _, child := range sortedChildren {
		typeDesc.Fields.List = append(typeDesc.Fields.List, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(child.name)},
			Type:  astTypeFromNode(child.node, opts),
			Tag:   astJSONTag(child.node.key, !child.node.required),
		})
	}

	return typeDesc
}

func astJSONTag(key string, omitempty bool) *ast.BasicLit {
	tag := fmt.Sprintf("%#v", key)
	tag = strings.Trim(tag, `"`)
	// if omitempty {
	// 	tag = fmt.Sprintf("`json:\"%s,omitempty\"`", tag)
	// } else {
	// 	tag = fmt.Sprintf("`json:\"%s\"`", tag)
	// }

	tag = fmt.Sprintf("`%s:\"%s\"`", tagName, tag)

	return &ast.BasicLit{
		Value: tag,
	}
}

func newEmptyInterfaceExpr() ast.Expr {
	return &ast.InterfaceType{
		Methods: &ast.FieldList{
			Opening: token.Pos(1),
			Closing: token.Pos(2),
		},
	}
}
