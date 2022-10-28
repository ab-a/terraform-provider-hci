package hci

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	hc "github.com/hypertec-cloud/go-hci"
	"github.com/hypertec-cloud/go-hci/services/hci"
)

// type of ACL rules
const (
	TCP  = "TCP"
	UDP  = "UDP"
	ICMP = "ICMP"
)

func resourceHciNetworkACLRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceHciNetworkACLRuleCreate,
		Update: resourceHciNetworkACLRuleUpdate,
		Read:   resourceHciNetworkACLRuleRead,
		Delete: resourceHciNetworkACLRuleDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"environment_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of environment where the network ACL rule should be created",
			},
			"rule_number": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The rule number of network ACL",
			},
			"cidr": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The network ACL rule cidr",
			},
			"action": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The network ACL rule action (i.e. Allow or Deny)",
				StateFunc: func(val interface{}) string {
					return strings.ToLower(val.(string))
				},
			},
			"protocol": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The network ACL rule protocol (i.e. TCP, UDP, ICMP or All)",
				StateFunc: func(val interface{}) string {
					return strings.ToLower(val.(string))
				},
			},
			"traffic_type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The network ACL rule traffc type (i.e. Ingress or Egress)",
				StateFunc: func(val interface{}) string {
					return strings.ToLower(val.(string))
				},
			},
			"icmp_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ICMP type. Can only be used with ICMP protocol.",
			},
			"icmp_code": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ICMP code. Can only be used with ICMP protocol.",
			},
			"start_port": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The start port. Can only be used with TCP/UDP protocol.",
			},
			"end_port": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The end port. Can only be used with TCP/UDP protocol.",
			},
			"network_acl_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Id of the network ACL of the network ACL rule",
			},
		},
	}
}

func resourceHciNetworkACLRuleCreate(d *schema.ResourceData, meta interface{}) error {
	hciResources, rerr := getResourcesForEnvironmentID(meta.(*hc.HciClient), d.Get("environment_id").(string))

	if rerr != nil {
		return rerr
	}
	aclRuleToCreate := hci.NetworkAclRule{
		RuleNumber:   d.Get("rule_number").(string),
		Cidr:         d.Get("cidr").(string),
		Action:       d.Get("action").(string),
		Protocol:     d.Get("protocol").(string),
		TrafficType:  d.Get("traffic_type").(string),
		NetworkAclId: d.Get("network_acl_id").(string),
	}
	fillPortFields(d, &aclRuleToCreate)
	fillIcmpFields(d, &aclRuleToCreate)
	if !(strings.EqualFold(TCP, aclRuleToCreate.Protocol) || strings.EqualFold(UDP, aclRuleToCreate.Protocol)) && (aclRuleToCreate.StartPort != "" || aclRuleToCreate.EndPort != "") {
		return fmt.Errorf("Cannot have ports if not TCP or UDP protocol")
	}
	if !strings.EqualFold(ICMP, aclRuleToCreate.Protocol) && (aclRuleToCreate.IcmpType != "" || aclRuleToCreate.IcmpCode != "") {
		return fmt.Errorf("Cannot have icmp fields if not ICMP protocol")
	}

	newACLRule, err := hciResources.NetworkAclRules.Create(aclRuleToCreate)
	if err != nil {
		return fmt.Errorf("Error creating the new network ACL rule %s: %s", aclRuleToCreate.RuleNumber, err)
	}
	d.SetId(newACLRule.Id)
	return resourceHciNetworkACLRuleRead(d, meta)
}

func resourceHciNetworkACLRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	hciResources, rerr := getResourcesForEnvironmentID(meta.(*hc.HciClient), d.Get("environment_id").(string))

	if rerr != nil {
		return rerr
	}
	aclRuleToUpdate := hci.NetworkAclRule{
		Id:          d.Id(),
		RuleNumber:  d.Get("rule_number").(string),
		Cidr:        d.Get("cidr").(string),
		Action:      d.Get("action").(string),
		Protocol:    d.Get("protocol").(string),
		TrafficType: d.Get("traffic_type").(string),
	}
	fillPortFields(d, &aclRuleToUpdate)
	fillIcmpFields(d, &aclRuleToUpdate)

	_, err := hciResources.NetworkAclRules.Update(d.Id(), aclRuleToUpdate)
	if err != nil {
		return err
	}
	return nil
}

func resourceHciNetworkACLRuleRead(d *schema.ResourceData, meta interface{}) error {
	hciResources, rerr := getResourcesForEnvironmentID(meta.(*hc.HciClient), d.Get("environment_id").(string))

	if rerr != nil {
		return rerr
	}
	aclRule, aErr := hciResources.NetworkAclRules.Get(d.Id())
	if aErr != nil {
		return handleNotFoundError("Network ACL rule", false, aErr, d)
	}

	if err := d.Set("rule_number", aclRule.RuleNumber); err != nil {
		return fmt.Errorf("Error reading Trigger: %s", err)
	}

	if err := d.Set("action", strings.ToLower(aclRule.Action)); err != nil {
		return fmt.Errorf("Error reading Trigger: %s", err)
	}

	if err := d.Set("protocol", strings.ToLower(aclRule.Protocol)); err != nil {
		return fmt.Errorf("Error reading Trigger: %s", err)
	}

	if err := d.Set("traffic_type", strings.ToLower(aclRule.TrafficType)); err != nil {
		return fmt.Errorf("Error reading Trigger: %s", err)
	}

	if err := d.Set("icmp_type", aclRule.IcmpType); err != nil {
		return fmt.Errorf("Error reading Trigger: %s", err)
	}

	if err := d.Set("icmp_code", aclRule.IcmpCode); err != nil {
		return fmt.Errorf("Error reading Trigger: %s", err)
	}

	if err := d.Set("start_port", aclRule.StartPort); err != nil {
		return fmt.Errorf("Error reading Trigger: %s", err)
	}

	if err := d.Set("end_port", aclRule.EndPort); err != nil {
		return fmt.Errorf("Error reading Trigger: %s", err)
	}

	if err := d.Set("network_acl_id", aclRule.NetworkAclId); err != nil {
		return fmt.Errorf("Error reading Trigger: %s", err)
	}

	return nil
}

func resourceHciNetworkACLRuleDelete(d *schema.ResourceData, meta interface{}) error {
	hciResources, rerr := getResourcesForEnvironmentID(meta.(*hc.HciClient), d.Get("environment_id").(string))

	if rerr != nil {
		return rerr
	}
	if _, err := hciResources.NetworkAclRules.Delete(d.Id()); err != nil {
		return handleNotFoundError("Network ACL rule", true, err, d)
	}
	return nil
}

func fillPortFields(d *schema.ResourceData, aclRule *hci.NetworkAclRule) {
	if v, ok := d.GetOk("start_port"); ok {
		aclRule.StartPort = v.(string)
	}
	if v, ok := d.GetOk("end_port"); ok {
		aclRule.EndPort = v.(string)
	}
}

func fillIcmpFields(d *schema.ResourceData, aclRule *hci.NetworkAclRule) {
	if v, ok := d.GetOk("icmp_type"); ok {
		aclRule.IcmpType = v.(string)
	}
	if v, ok := d.GetOk("icmp_code"); ok {
		aclRule.IcmpCode = v.(string)
	}
}
