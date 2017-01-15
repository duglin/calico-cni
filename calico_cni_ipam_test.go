package main_test

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/projectcalico/cni-plugin/test_utils"
	"github.com/projectcalico/libcalico-go/lib/testutils"
)

var plugin = "calico-ipam"

var _ = Describe("Calico IPAM Tests", func() {
	BeforeEach(func() {
		WipeEtcd()
		testutils.CreateNewIPPool(*calicoClient, "192.168.0.0/16", false, false, true)
		testutils.CreateNewIPPool(*calicoClient, "fd80:24e2:f998:72d6::/64", false, false, true)
	})

	Describe("Run IPAM plugin", func() {
		Context("Do it", func() {
			DescribeTable("Request different numbers of IP addresses",
				func(expectedIPv4, expectedIPv6 bool, netconf string) {

					result, _ := RunIPAMPlugin(netconf, "ADD", "")

					if expectedIPv4 {
						Expect(result.IP4.IP.Mask.String()).Should(Equal("ffffffff"))
					}

					if expectedIPv6 {
						Expect(result.IP6.IP.Mask.String()).Should(Equal("ffffffffffffffffffffffffffffffff"))
					}

					// I can't find any testable side effects for this
					_, _ = RunIPAMPlugin(netconf, "DEL", "")
				},
				Entry("IPAM with no configuration", true, false, fmt.Sprintf(`
			{
			  "name": "net1",
			  "type": "calico",
			  "etcd_endpoints": "http://%s:2379",
			  "ipam": {
			    "type": "%s"
			  }
			}`, os.Getenv("ETCD_IP"), plugin)),
				Entry("IPAM with IPv4 (explicit)", true, false, fmt.Sprintf(`
			{
			  "name": "net1",
			  "type": "calico",
			  "etcd_endpoints": "http://%s:2379",
			  "ipam": {
			    "type": "%s",
			    "assign_ipv4": "true"
			  }
			}`, os.Getenv("ETCD_IP"), plugin)),
				Entry("IPAM with IPv6 only", false, true, fmt.Sprintf(`
			{
			  "name": "net1",
			  "type": "calico",
			  "etcd_endpoints": "http://%s:2379",
			  "ipam": {
			    "type": "%s",
			    "assign_ipv4": "false",
			    "assign_ipv6": "true"
			  }
			}`, os.Getenv("ETCD_IP"), plugin)),
				Entry("IPAM with IPv4 and IPv6", true, true, fmt.Sprintf(`
			{
			  "name": "net1",
			  "type": "calico",
			  "etcd_endpoints": "http://%s:2379",
			  "ipam": {
			    "type": "%s",
			    "assign_ipv4": "true",
			    "assign_ipv6": "true"
			  }
			}`, os.Getenv("ETCD_IP"), plugin)),
			)
		})
	})

	Describe("Run IPAM plugin - Verify IP Pools", func() {
		Context("Pass valid pools", func() {
			It("successfully networks the namespace", func() {
				netconf := fmt.Sprintf(`
			        {
			          "name": "net1",
			          "type": "calico",
			          "etcd_endpoints": "http://%s:2379",
			          "ipam": {
			            "type": "%s",
			            "assign_ipv4": "true",
						"ipv4_pools": [ "192.168.0.0/16" ]
			          }
			        }`, os.Getenv("ETCD_IP"), plugin)
				_, _, _, _, _, _, err := CreateContainer(netconf)
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		Context("Pass more than one pool", func() {
			It("successfully networks the namespace", func() {
				PreCreatePool("192.169.1.0/24")
				netconf := fmt.Sprintf(`
			        {
			          "name": "net1",
			          "type": "calico",
			          "etcd_endpoints": "http://%s:2379",
			          "ipam": {
			            "type": "%s",
			            "assign_ipv4": "true",
						"ipv4_pools": [ "192.169.1.0/24", "192.168.0.0/16" ]
			          }
			        }`, os.Getenv("ETCD_IP"), plugin)
				_, _, session, _, _, _, err := CreateContainer(netconf)
				if err != nil {
					fmt.Printf("Session Err: %v\n", string(session.Err.Contents()))
				}
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		Context("Pass an invalid pool", func() {
			It("fails to network the namespace", func() {
				// Put the bogus pool last in the array
				netconf := fmt.Sprintf(`
			        {
			          "name": "net1",
			          "type": "calico",
			          "etcd_endpoints": "http://%s:2379",
			          "ipam": {
			            "type": "%s",
			            "assign_ipv4": "true",
						"ipv4_pools": [ "192.168.0.0/16", "192.169.1.0/24" ]
			          }
			        }`, os.Getenv("ETCD_IP"), plugin)
				_, _, session, _, _, _, err := CreateContainer(netconf)
				Expect(err).Should(HaveOccurred())
				Expect(session.Err.Contents()).Should(ContainSubstring("192.169.1.0/24) does not exist"))
			})

			It("fails to network the namespace", func() {
				// Put the bogus pool first in the array
				netconf := fmt.Sprintf(`
			        {
			          "name": "net1",
			          "type": "calico",
			          "etcd_endpoints": "http://%s:2379",
			          "ipam": {
			            "type": "%s",
			            "assign_ipv4": "true",
						"ipv4_pools": [ "192.169.1.0/24", "192.168.0.0/16" ]
			          }
			        }`, os.Getenv("ETCD_IP"), plugin)
				_, _, session, _, _, _, err := CreateContainer(netconf)
				Expect(err).Should(HaveOccurred())
				Expect(session.Err.Contents()).Should(ContainSubstring("192.169.1.0/24) does not exist"))
			})
		})

	})

	Describe("Run IPAM plugin", func() {
		netconf := fmt.Sprintf(`
					{"name": "net1",
					  "type": "calico",
					  "etcd_endpoints": "http://%s:2379",
					  "ipam": {
					    "type": "%s"
					  }
					}`, os.Getenv("ETCD_IP"), plugin)
		Context("Pass explicit IP address", func() {
			It("Return the expected IP", func() {
				result, _ := RunIPAMPlugin(netconf, "ADD", "IP=192.168.123.123")
				Expect(result.IP4.IP.String()).Should(Equal("192.168.123.123/32"))
			})
			It("Return the expected IP twice after deleting in the middle", func() {
				result, _ := RunIPAMPlugin(netconf, "ADD", "IP=192.168.123.123")
				Expect(result.IP4.IP.String()).Should(Equal("192.168.123.123/32"))
				_, _ = RunIPAMPlugin(netconf, "DEL", "IP=192.168.123.123")
				result, _ = RunIPAMPlugin(netconf, "ADD", "IP=192.168.123.123")
				Expect(result.IP4.IP.String()).Should(Equal("192.168.123.123/32"))
			})
			It("Doesn't allow an explicit IP to be assigned twice", func() {
				result, _ := RunIPAMPlugin(netconf, "ADD", "IP=192.168.123.123")
				Expect(result.IP4.IP.String()).Should(Equal("192.168.123.123/32"))
				result, exitCode := RunIPAMPlugin(netconf, "ADD", "IP=192.168.123.123")
				Expect(exitCode).Should(BeNumerically(">", 0))
			})
		})
	})
})
